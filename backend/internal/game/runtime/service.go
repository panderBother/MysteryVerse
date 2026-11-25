package runtime

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/compile"
	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/game/ports"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	scriptservice "github.com/mysteryverse/mysteryverse/backend/internal/script"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type Service struct {
	repo     storage.Repository
	roomPort ports.RoomStartPort
	compiler *compile.Compiler
	loader   *scriptservice.Loader
	cipher   *secret.Cipher
	ids      id.Generator
	now      func() time.Time
}

func NewService(
	repo storage.Repository,
	roomPort ports.RoomStartPort,
	cipher *secret.Cipher,
) *Service {
	return &Service{
		repo:     repo,
		roomPort: roomPort,
		compiler: compile.NewCompiler(),
		loader:   scriptservice.NewLoader(),
		cipher:   cipher,
		ids:      id.ULIDGenerator{},
		now:      time.Now,
	}
}

func (s *Service) Start(
	ctx context.Context,
	actorUserID string,
	roomID string,
	input StartInput,
	idempotencyKey string,
) (StartResult, error) {
	if idempotencyKey == "" {
		return StartResult{}, validationError("Idempotency-Key is required")
	}
	if !input.RulesAck {
		return StartResult{}, validationError("rules_ack must be true")
	}
	isHost, err := s.roomPort.IsHost(ctx, s.repo, roomID, actorUserID)
	if err != nil || !isHost {
		return StartResult{}, hostRequiredError()
	}
	if existing, err := s.repo.GetGameSessionByRoom(ctx, roomID); err == nil {
		return StartResult{
			GameID:        existing.ID,
			RoomStatus:    "in_game",
			StateVersion:  existing.StateVersion,
			GameEventSeq:  existing.CurrentEventSeq,
			EngineVersion: existing.EngineVersion,
			GameType:      existing.GameType,
			Replay:        true,
		}, nil
	} else if !isStorageCode(err, database.CodeNotFound) {
		return StartResult{}, mapGameStorageError(err)
	}

	var result StartResult
	err = s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		startAt := s.now().UTC()
		snapshot, err := s.roomPort.LockForStart(
			ctx,
			tx,
			actorUserID,
			roomID,
			input.ExpectedRoomEventSeq,
			startAt,
		)
		if err != nil {
			return err
		}
		if input.ContentVersionID != "" && input.ContentVersionID != snapshot.ScriptVersionID {
			return apperror.New(
				"CONTENT_VERSION_UNAVAILABLE",
				"The requested content version is not the room's locked version.",
				false,
				nil,
			)
		}
		loaded, err := s.loader.LoadPlayable(
			ctx,
			tx,
			snapshot.ScriptVersionID,
			scriptservice.LoadModeStart,
		)
		if err != nil {
			return apperror.New(
				"CONTENT_UNAVAILABLE",
				"The script cannot be compiled for play.",
				false,
				nil,
			)
		}
		if snapshot.RuleTemplateID != nil &&
			(loaded.Version.RuleTemplateID == nil ||
				*snapshot.RuleTemplateID != *loaded.Version.RuleTemplateID) {
			return apperror.New(
				"CONTENT_UNAVAILABLE",
				"The room rule template does not match the published script version.",
				false,
				nil,
			)
		}
		compiled, err := s.compiler.Compile(loaded)
		if err != nil {
			return apperror.New(
				"CONTENT_UNAVAILABLE",
				"The script cannot be compiled for play.",
				false,
				map[string]string{"reason": err.Error()},
			)
		}
		if len(snapshot.Members) != len(compiled.Config.Characters) {
			return apperror.New(
				"PLAYER_COUNT_MISMATCH",
				"The room player count does not match the script roles.",
				false,
				map[string]int{
					"players":    len(snapshot.Members),
					"characters": len(compiled.Config.Characters),
				},
			)
		}

		gameID := s.ids.New()
		players := make([]domain.Player, 0, len(snapshot.Members))
		for _, member := range snapshot.Members {
			players = append(players, domain.Player{
				ID:                 s.ids.New(),
				UserID:             member.UserID,
				RoomMemberID:       member.RoomMemberID,
				SeatNo:             member.SeatNo,
				DisplayName:        member.DisplayName,
				RuntimeStatus:      "active",
				SearchQuotaByPhase: make(map[string]int),
			})
		}
		sort.Slice(players, func(i int, j int) bool {
			return players[i].SeatNo < players[j].SeatNo
		})

		engine, err := domain.NewScriptMurderEngine(compiled.Config)
		if err != nil {
			return internalError()
		}
		roles, err := engine.AssignRoles(players, compiled.Config, cryptoRandomizer{})
		if err != nil {
			return apperror.New(
				"CONTENT_UNAVAILABLE",
				"Role assignment failed.",
				false,
				nil,
			)
		}
		for index := range roles {
			roles[index].ID = s.ids.New()
		}

		firstPhase := compiled.Config.Phases[0]
		state := domain.RuntimeState{
			GameID:          gameID,
			GameType:        "script_murder",
			Status:          domain.StatusRunning,
			StateVersion:    1,
			PhaseIndex:      0,
			PhaseKey:        firstPhase.Key,
			PhaseStartedAt:  startAt,
			Players:         players,
			Roles:           roles,
			SearchPointUses: make(map[string]int),
		}
		if firstPhase.TimeoutSeconds != nil && *firstPhase.TimeoutSeconds > 0 {
			deadline := startAt.Add(time.Duration(*firstPhase.TimeoutSeconds) * time.Second)
			state.PhaseDeadlineAt = &deadline
		}
		stateHash := domain.StateHash(state)

		if err := tx.CreateGameSession(ctx, sqlc.CreateGameSessionParams{
			ID:                gameID,
			RoomID:            roomID,
			GameType:          "script_murder",
			ScriptVersionID:   snapshot.ScriptVersionID,
			RuleTemplateID:    nullableStringParam(snapshot.RuleTemplateID),
			EngineKey:         domain.EngineKey,
			EngineVersion:     domain.EngineVersion,
			PackageUri:        "script-version://" + snapshot.ScriptVersionID,
			PackageHash:       compiled.PackageHash,
			SchemaVersion:     compiled.Config.SchemaVersion,
			Status:            "running",
			StateVersion:      0,
			CurrentEventSeq:   0,
			StateHash:         stateHash,
			OwnerFencingToken: 0,
			StartedAt:         startAt,
		}); err != nil {
			return mapGameStorageError(err)
		}

		for _, player := range players {
			if err := tx.CreateGamePlayer(ctx, sqlc.CreateGamePlayerParams{
				ID:                  player.ID,
				GameSessionID:       gameID,
				RoomMemberID:        player.RoomMemberID,
				UserID:              sql.NullString{String: player.UserID, Valid: true},
				SeatNo:              uint8(player.SeatNo),
				ParticipantKind:     "human",
				DisplayNameSnapshot: player.DisplayName,
				RuntimeStatus:       "active",
				JoinedAt:            startAt,
			}); err != nil {
				return mapGameStorageError(err)
			}
		}

		for _, role := range roles {
			secretID, err := s.storeSecret(ctx, tx, secretRecordInput{
				ScopeType:       "game",
				ScopeID:         gameID,
				OwnerType:       "role",
				OwnerID:         role.ID,
				PurposeCode:     "private_role",
				VersionNo:       1,
				Sensitivity:     3,
				VisibilityScope: "player",
				Value:           role,
				AAD:             "game:" + gameID + ":role:" + role.ID,
			})
			if err != nil {
				return err
			}
			if err := tx.CreateGameRole(ctx, sqlc.CreateGameRoleParams{
				ID:                role.ID,
				GameSessionID:     gameID,
				RoleInstanceKey:   role.CharacterKey,
				PlayerID:          sql.NullString{String: role.PlayerID, Valid: true},
				SourceCharacterID: sql.NullString{String: role.CharacterID, Valid: true},
				FactionKey:        role.FactionKey,
				TeamKey:           sql.NullString{String: role.TeamKey, Valid: role.TeamKey != ""},
				SecretPayloadID:   secretID,
				AssignedAt:        startAt,
			}); err != nil {
				return mapGameStorageError(err)
			}
		}

		events := buildStartEvents(gameID, actorUserID, startAt, roles, firstPhase)
		finalSeq := uint64(0)
		for _, event := range events {
			finalSeq++
			if _, err := s.persistDomainEvent(
				ctx,
				tx,
				gameID,
				finalSeq,
				event,
				"user",
				actorUserID,
				"",
				0,
				1,
				startAt,
			); err != nil {
				return err
			}
		}

		state.EventSeq = finalSeq
		stateHash = domain.StateHash(state)
		stateSecretID, err := s.storeSecret(ctx, tx, secretRecordInput{
			ScopeType:       "game",
			ScopeID:         gameID,
			OwnerType:       "game_state",
			OwnerID:         gameID,
			PurposeCode:     "runtime_secret",
			VersionNo:       1,
			Sensitivity:     3,
			VisibilityScope: "runtime",
			Value:           state,
			AAD:             "game:" + gameID + ":state",
		})
		if err != nil {
			return err
		}
		publicJSON, _ := json.Marshal(publicState(state, compiled.Config))
		if err := tx.CreateGameState(ctx, sqlc.CreateGameStateParams{
			ID:                  s.ids.New(),
			GameSessionID:       gameID,
			PublicStateJson:     publicJSON,
			SecretPayloadID:     stateSecretID,
			SchemaVersion:       compiled.Config.SchemaVersion,
			PhaseKey:            state.PhaseKey,
			PhaseStartedAt:      sql.NullTime{Time: state.PhaseStartedAt, Valid: true},
			PhaseDeadlineAt:     nullableTimeParam(state.PhaseDeadlineAt),
			StateHash:           stateHash,
			LastEventSeq:        finalSeq,
			MaterializerVersion: "murder-state-v1",
		}); err != nil {
			return mapGameStorageError(err)
		}
		if err := tx.UpdateGameSessionCommand(ctx, sqlc.UpdateGameSessionCommandParams{
			Status:          "running",
			StateVersion:    1,
			CurrentEventSeq: finalSeq,
			StateHash:       stateHash,
			ID:              gameID,
			StateVersion_2:  0,
		}); err != nil {
			return mapGameStorageError(err)
		}
		if state.PhaseDeadlineAt != nil {
			if err := s.createTimer(ctx, tx, gameID, state.PhaseKey, *state.PhaseDeadlineAt, 1); err != nil {
				return err
			}
		}
		if err := s.roomPort.MarkStarted(
			ctx,
			tx,
			roomID,
			gameID,
			input.ExpectedRoomEventSeq,
			startAt,
		); err != nil {
			return err
		}

		result = StartResult{
			GameID:        gameID,
			RoomStatus:    "in_game",
			StateVersion:  1,
			GameEventSeq:  finalSeq,
			EngineVersion: domain.EngineVersion,
			GameType:      "script_murder",
		}
		return nil
	})
	if err != nil {
		return StartResult{}, err
	}
	return result, nil
}

func (s *Service) Projection(
	ctx context.Context,
	actorUserID string,
	gameID string,
) (Projection, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return Projection{}, mapGameStorageError(err)
	}
	player, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	})
	if err != nil {
		if isStorageCode(err, database.CodeNotFound) {
			return Projection{}, notGamePlayerError()
		}
		return Projection{}, mapGameStorageError(err)
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return Projection{}, err
	}
	role := roleForPlayer(state.Roles, player.ID)
	phase := compiled.Config.Phases[state.PhaseIndex]
	isHost, _ := s.roomPort.IsHost(ctx, s.repo, session.RoomID, actorUserID)
	domainPlayer := domainPlayerForID(state.Players, player.ID)
	actions := availableActions(state, compiled.Config, domainPlayer, role, isHost)

	return Projection{
		GameID:       session.ID,
		GameType:     session.GameType,
		Status:       string(state.Status),
		StateVersion: session.StateVersion,
		EventSeq:     session.CurrentEventSeq,
		Viewer:       Viewer{PlayerID: player.ID, RoleID: role.ID},
		Phase: PhaseView{
			Key:        phase.Key,
			Type:       string(phase.Kind),
			Title:      phase.Title,
			StartedAt:  state.PhaseStartedAt,
			DeadlineAt: state.PhaseDeadlineAt,
			Paused:     state.Status == domain.StatusPaused,
		},
		Players:            playerViews(state.Players),
		PublicState:        publicState(state, compiled.Config),
		PrivateState:       privateState(role, state, compiled.Config),
		AvailableActions:   actions,
		ViewerCapabilities: projectionCapabilities(state.Status, isHost),
	}, nil
}

func (s *Service) loadRuntime(
	ctx context.Context,
	repo storage.Repository,
	session sqlc.GameSession,
) (domain.RuntimeState, compile.Result, error) {
	gameState, err := repo.GetGameState(ctx, session.ID)
	if err != nil {
		return domain.RuntimeState{}, compile.Result{}, mapGameStorageError(err)
	}
	payload, err := repo.GetSecretPayload(ctx, gameState.SecretPayloadID)
	if err != nil {
		return domain.RuntimeState{}, compile.Result{}, internalError()
	}
	plaintext, err := s.cipher.Open(secret.Sealed{
		KeyID:       payload.KeyID,
		Nonce:       payload.Nonce,
		Ciphertext:  payload.Ciphertext,
		ContentHash: payload.ContentHash,
	}, "game:"+session.ID+":state")
	if err != nil {
		return domain.RuntimeState{}, compile.Result{}, internalError()
	}
	var state domain.RuntimeState
	if err := json.Unmarshal(plaintext, &state); err != nil {
		return domain.RuntimeState{}, compile.Result{}, internalError()
	}
	loaded, err := s.loader.LoadPlayable(
		ctx,
		repo,
		session.ScriptVersionID,
		scriptservice.LoadModeRuntime,
	)
	if err != nil {
		return domain.RuntimeState{}, compile.Result{}, apperror.New(
			"CONTENT_UNAVAILABLE",
			"The game content version cannot be loaded.",
			false,
			nil,
		)
	}
	compiled, err := s.compiler.Compile(loaded)
	if err != nil {
		return domain.RuntimeState{}, compile.Result{}, apperror.New(
			"CONTENT_UNAVAILABLE",
			"The game content version cannot be loaded.",
			false,
			nil,
		)
	}
	return state, compiled, nil
}

func (s *Service) storeSecret(
	ctx context.Context,
	repo storage.Repository,
	input secretRecordInput,
) (string, error) {
	raw, err := json.Marshal(input.Value)
	if err != nil {
		return "", internalError()
	}
	sealed, err := s.cipher.Seal(raw, input.AAD)
	if err != nil {
		return "", internalError()
	}
	idValue := s.ids.New()
	if err := repo.CreateSecretPayload(ctx, sqlc.CreateSecretPayloadParams{
		ID:              idValue,
		ScopeType:       input.ScopeType,
		ScopeID:         input.ScopeID,
		OwnerType:       input.OwnerType,
		OwnerID:         input.OwnerID,
		PurposeCode:     input.PurposeCode,
		VersionNo:       secretVersion(input.VersionNo),
		Sensitivity:     input.Sensitivity,
		VisibilityScope: input.VisibilityScope,
		SchemaVersion:   "1.0.0",
		KeyID:           sealed.KeyID,
		Nonce:           sealed.Nonce,
		Ciphertext:      sealed.Ciphertext,
		ContentHash:     sealed.ContentHash,
		RetentionClass:  "game_lifetime",
	}); err != nil {
		return "", mapGameStorageError(err)
	}
	return idValue, nil
}

func (s *Service) decryptRole(
	ctx context.Context,
	repo storage.Repository,
	role sqlc.GameRole,
) (domain.Role, error) {
	payload, err := repo.GetSecretPayload(ctx, role.SecretPayloadID)
	if err != nil {
		return domain.Role{}, internalError()
	}
	plaintext, err := s.cipher.Open(secret.Sealed{
		KeyID:       payload.KeyID,
		Nonce:       payload.Nonce,
		Ciphertext:  payload.Ciphertext,
		ContentHash: payload.ContentHash,
	}, "game:"+role.GameSessionID+":role:"+role.ID)
	if err != nil {
		return domain.Role{}, internalError()
	}
	var value domain.Role
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return domain.Role{}, internalError()
	}
	return value, nil
}

type secretRecordInput struct {
	ScopeType       string
	ScopeID         string
	OwnerType       string
	OwnerID         string
	PurposeCode     string
	VersionNo       uint32
	Sensitivity     uint8
	VisibilityScope string
	Value           any
	AAD             string
}

func secretVersion(value uint32) uint32 {
	if value == 0 {
		return 1
	}
	return value
}

type cryptoRandomizer struct{}

func (cryptoRandomizer) Permutation(size int) ([]int, error) {
	permutation := make([]int, size)
	for index := range permutation {
		permutation[index] = index
	}
	for index := size - 1; index > 0; index-- {
		var randomByte [8]byte
		if _, err := rand.Read(randomByte[:]); err != nil {
			return nil, err
		}
		var value uint64
		for _, item := range randomByte {
			value = value<<8 | uint64(item)
		}
		swap := int(value % uint64(index+1))
		permutation[index], permutation[swap] = permutation[swap], permutation[index]
	}
	return permutation, nil
}

func buildStartEvents(
	gameID string,
	hostUserID string,
	startedAt time.Time,
	roles []domain.Role,
	firstPhase domain.Phase,
) []domain.DomainEvent {
	events := []domain.DomainEvent{{
		EventType:       "game.started",
		EventCategory:   "incident",
		VisibilityScope: "room",
		Sensitivity:     0,
		Payload: map[string]any{
			"schema_version": "1.0.0",
			"game_id":        gameID,
			"engine_version": domain.EngineVersion,
		},
		StateChanged: true,
	}}
	for _, role := range roles {
		events = append(events, domain.DomainEvent{
			EventType:       "game.role_assigned",
			EventCategory:   "adjudication",
			VisibilityScope: "player",
			Visibility:      map[string]any{"players": []string{role.PlayerID}},
			Sensitivity:     3,
			Payload: map[string]any{
				"schema_version": "1.0.0",
				"role_id":        role.ID,
			},
			StateChanged: true,
		})
	}
	events = append(events, domain.DomainEvent{
		EventType:       "game.phase_started",
		EventCategory:   "phase",
		VisibilityScope: "room",
		Sensitivity:     0,
		Payload: map[string]any{
			"schema_version": "1.0.0",
			"phase_key":      firstPhase.Key,
			"phase_type":     firstPhase.Kind,
		},
		StateChanged: true,
	})
	return events
}

func publicState(state domain.RuntimeState, config domain.ScriptConfig) map[string]any {
	phase := config.Phases[state.PhaseIndex]
	ready := 0
	for _, player := range state.Players {
		if player.ReadingCompleted {
			ready++
		}
	}
	return map[string]any{
		"schema_version": "1.0.0",
		"status":         state.Status,
		"phase": map[string]any{
			"key":         phase.Key,
			"type":        phase.Kind,
			"title":       phase.Title,
			"started_at":  state.PhaseStartedAt,
			"deadline_at": state.PhaseDeadlineAt,
			"paused":      state.Status == domain.StatusPaused,
		},
		"progress": map[string]any{
			"players":           len(state.Players),
			"reading_completed": ready,
		},
	}
}

func privateState(
	role domain.Role,
	state domain.RuntimeState,
	config domain.ScriptConfig,
) map[string]any {
	key := "quota:" + role.PlayerID + ":" + state.PhaseKey
	used := state.SearchPointUses[key]
	return map[string]any{
		"role": map[string]any{
			"role_id":        role.ID,
			"character_id":   role.CharacterID,
			"display_name":   role.DisplayName,
			"faction":        role.FactionKey,
			"team":           role.TeamKey,
			"public_profile": role.PublicProfile,
		},
		"search_quota": map[string]int{
			"used":      used,
			"remaining": max(config.Policy.DefaultSearchQuota-used, 0),
		},
	}
}

func playerViews(players []domain.Player) []PlayerView {
	views := make([]PlayerView, 0, len(players))
	for _, player := range players {
		views = append(views, PlayerView{
			PlayerID:        player.ID,
			SeatNo:          player.SeatNo,
			DisplayName:     player.DisplayName,
			Status:          player.RuntimeStatus,
			ReadingComplete: player.ReadingCompleted,
		})
	}
	return views
}

func projectionCapabilities(status domain.Status, isHost bool) []string {
	capabilities := []string{"view_role", "view_clues"}
	if isHost {
		capabilities = append(capabilities, "pause_game")
	}
	if status == domain.StatusPaused && isHost {
		capabilities = append(capabilities, "resume_game")
	}
	return capabilities
}

func roleForPlayer(roles []domain.Role, playerID string) domain.Role {
	for _, role := range roles {
		if role.PlayerID == playerID {
			return role
		}
	}
	return domain.Role{}
}

func domainPlayerForID(players []domain.Player, playerID string) domain.Player {
	for _, player := range players {
		if player.ID == playerID {
			return player
		}
	}
	return domain.Player{}
}
