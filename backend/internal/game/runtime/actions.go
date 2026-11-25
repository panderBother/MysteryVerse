package runtime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func (s *Service) ReadingComplete(
	ctx context.Context,
	actorUserID string,
	gameID string,
	expectedVersion uint64,
	idempotencyKey string,
) (ActionResult, error) {
	return s.applyAction(ctx, actorUserID, gameID, expectedVersion, idempotencyKey, domain.Action{
		Type:    domain.ActionReadingComplete,
		Payload: map[string]any{},
	})
}

func (s *Service) Search(
	ctx context.Context,
	actorUserID string,
	gameID string,
	input SearchInput,
	idempotencyKey string,
) (ActionResult, error) {
	if input.SearchPointID == "" {
		return ActionResult{}, validationError("search_point_id is required")
	}
	return s.applyAction(
		ctx,
		actorUserID,
		gameID,
		input.ExpectedVersion,
		idempotencyKey,
		domain.Action{
			Type: domain.ActionSearch,
			Payload: map[string]any{
				"search_point_id":   input.SearchPointID,
				"client_request_id": input.ClientRequestID,
			},
		},
	)
}

func (s *Service) Vote(
	ctx context.Context,
	actorUserID string,
	gameID string,
	input VoteInput,
	idempotencyKey string,
) (ActionResult, error) {
	return s.applyAction(
		ctx,
		actorUserID,
		gameID,
		input.ExpectedVersion,
		idempotencyKey,
		domain.Action{
			Type: domain.ActionVote,
			Payload: map[string]any{
				"target_player_id":  input.TargetPlayerID,
				"abstain":           input.Abstain,
				"round_key":         input.RoundKey,
				"client_request_id": input.ClientRequestID,
			},
		},
	)
}

func (s *Service) AdvancePhase(
	ctx context.Context,
	actorUserID string,
	gameID string,
	expectedVersion uint64,
	idempotencyKey string,
) (ActionResult, error) {
	return s.applyAction(ctx, actorUserID, gameID, expectedVersion, idempotencyKey, domain.Action{
		Type:    domain.ActionAdvancePhase,
		Payload: map[string]any{},
	})
}

func (s *Service) applyAction(
	ctx context.Context,
	actorUserID string,
	gameID string,
	expectedVersion uint64,
	idempotencyKey string,
	action domain.Action,
) (ActionResult, error) {
	if idempotencyKey == "" {
		return ActionResult{}, validationError("Idempotency-Key is required")
	}

	var result ActionResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		session, err := tx.GetGameSessionForUpdate(ctx, gameID)
		if err != nil {
			return mapGameStorageError(err)
		}
		gamePlayer, err := tx.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
			GameSessionID: gameID,
			UserID:        sql.NullString{String: actorUserID, Valid: true},
		})
		if err != nil {
			if isStorageCode(err, database.CodeNotFound) {
				return notGamePlayerError()
			}
			return mapGameStorageError(err)
		}
		payloadJSON, _ := json.Marshal(action)
		payloadHash := sha256.Sum256(payloadJSON)
		intent, replay, err := s.findOrCreateIntent(
			ctx,
			tx,
			session,
			gamePlayer.ID,
			action.Type,
			idempotencyKey,
			payloadJSON,
			payloadHash[:],
			expectedVersion,
		)
		if err != nil {
			return err
		}
		if replay {
			result = resultFromIntent(intent, session.StateVersion)
			return nil
		}
		if session.Status == "completed" || session.Status == "aborted" {
			return apperror.New("GAME_ALREADY_ENDED", "The game has already ended.", false, nil)
		}
		if expectedVersion != session.StateVersion {
			return versionConflictError()
		}
		if _, err := tx.GetGameStateForUpdate(ctx, gameID); err != nil {
			return mapGameStorageError(err)
		}
		state, compiled, err := s.loadRuntime(ctx, tx, session)
		if err != nil {
			return err
		}
		role := roleForPlayer(state.Roles, gamePlayer.ID)
		if role.ID == "" {
			return apperror.New("ROLE_NOT_BOUND", "The player has no active role binding.", false, nil)
		}

		engine, err := domain.NewScriptMurderEngine(compiled.Config)
		if err != nil {
			return internalError()
		}
		isHost, _ := s.roomPort.IsHost(ctx, tx, session.RoomID, actorUserID)
		actor := domain.Actor{
			UserID:   actorUserID,
			PlayerID: gamePlayer.ID,
			RoleID:   role.ID,
			IsHost:   isHost,
		}
		now := s.now().UTC()
		decision, err := engine.Apply(state, compiled.Config, actor, action, now)
		if err != nil {
			return mapDomainActionError(err)
		}
		if decision.State.Status == domain.StatusSettling {
			end, completed, err := engine.CheckEnd(decision.State, compiled.Config)
			if err != nil || !completed {
				return internalError()
			}
			decision.Events = append(decision.Events, domain.DomainEvent{
				EventType:       "game.settled",
				EventCategory:   "adjudication",
				VisibilityScope: "room",
				Sensitivity:     0,
				Payload: map[string]any{
					"schema_version": "1.0.0",
					"result_code":    end.ResultCode,
					"winner_team":    end.WinnerTeam,
					"summary":        end.Summary,
				},
				StateChanged: true,
			})
			decision.State.Status = domain.StatusCompleted
			decision.State.ResultCode = end.ResultCode
			decision.State.SettlementSummary = end.Summary
			result.ResultCode = end.ResultCode
			result.Summary = end.Summary
		}

		newVersion := session.StateVersion + 1
		seq := session.CurrentEventSeq
		eventIDs := make([]string, 0, len(decision.Events))
		for _, event := range decision.Events {
			seq++
			eventID, err := s.persistDomainEvent(
				ctx,
				tx,
				gameID,
				seq,
				event,
				"player",
				gamePlayer.ID,
				intent.ID,
				session.StateVersion,
				newVersion,
				now,
			)
			if err != nil {
				return err
			}
			eventIDs = append(eventIDs, eventID)
		}
		decision.State.StateVersion = newVersion
		decision.State.EventSeq = seq
		stateHash := domain.StateHash(decision.State)

		switch action.Type {
		case domain.ActionSearch:
			if err := persistSearchClaim(ctx, tx, s.ids.New(), gameID, gamePlayer.ID, intent.ID, eventIDs[0], decision, now); err != nil {
				return err
			}
		case domain.ActionVote:
			targetID, _ := action.Payload["target_player_id"].(string)
			targetRoleID := sql.NullString{}
			if targetID != "" {
				targetRole, err := tx.GetGameRoleByPlayer(ctx, sqlc.GetGameRoleByPlayerParams{
					GameSessionID: gameID,
					PlayerID:      sql.NullString{String: targetID, Valid: true},
				})
				if err == nil {
					targetRoleID = sql.NullString{String: targetRole.ID, Valid: true}
				}
			}
			if _, err := tx.UpsertGameVote(ctx, sqlc.UpsertGameVoteParams{
				ID:             s.ids.New(),
				GameSessionID:  gameID,
				RoundKey:       decision.State.PhaseRoundKey,
				VoterPlayerID:  gamePlayer.ID,
				TargetPlayerID: sql.NullString{String: targetID, Valid: targetID != ""},
				TargetRoleID:   targetRoleID,
				IsAbstain:      boolValue(action.Payload["abstain"]),
				IntentID:       sql.NullString{String: intent.ID, Valid: true},
				EventID:        sql.NullString{String: eventIDs[0], Valid: true},
				CastAt:         now,
			}); err != nil {
				return mapGameStorageError(err)
			}
		}

		stateSecretID, err := s.storeSecret(ctx, tx, secretRecordInput{
			ScopeType:       "game",
			ScopeID:         gameID,
			OwnerType:       "game_state",
			OwnerID:         gameID,
			PurposeCode:     "runtime_secret",
			VersionNo:       uint32(decision.State.StateVersion),
			Sensitivity:     3,
			VisibilityScope: "runtime",
			Value:           decision.State,
			AAD:             "game:" + gameID + ":state",
		})
		if err != nil {
			return err
		}
		publicJSON, _ := json.Marshal(publicState(decision.State, compiled.Config))
		if err := tx.UpdateGameState(ctx, sqlc.UpdateGameStateParams{
			PublicStateJson:     publicJSON,
			SecretPayloadID:     stateSecretID,
			SchemaVersion:       compiled.Config.SchemaVersion,
			PhaseKey:            decision.State.PhaseKey,
			PhaseStartedAt:      nullableTimeParam(timePointer(decision.State.PhaseStartedAt)),
			PhaseDeadlineAt:     nullableTimeParam(decision.State.PhaseDeadlineAt),
			StateHash:           stateHash,
			LastEventSeq:        seq,
			MaterializerVersion: "murder-state-v1",
			GameSessionID:       gameID,
		}); err != nil {
			return mapGameStorageError(err)
		}

		status := string(decision.State.Status)
		endedAt := sql.NullTime{}
		settleReason := sql.NullString{}
		if decision.State.Status == domain.StatusCompleted {
			endedAt = sql.NullTime{Time: now, Valid: true}
			settleReason = sql.NullString{String: result.ResultCode, Valid: result.ResultCode != ""}
		}
		if err := tx.UpdateGameSessionCommand(ctx, sqlc.UpdateGameSessionCommandParams{
			Status:          status,
			StateVersion:    newVersion,
			CurrentEventSeq: seq,
			StateHash:       stateHash,
			EndedAt:         endedAt,
			SettleReason:    settleReason,
			ID:              gameID,
			StateVersion_2:  session.StateVersion,
		}); err != nil {
			if errors.Is(err, storage.ErrNoRowsAffected) {
				return versionConflictError()
			}
			return mapGameStorageError(err)
		}
		if decision.State.Status == domain.StatusCompleted {
			end := domain.EndResult{
				ResultCode: result.ResultCode,
				WinnerTeam: stringValue(result.Summary["winner_team"]),
				Summary:    result.Summary,
			}
			if err := s.createResult(ctx, tx, gameID, newVersion, seq, end, now); err != nil {
				return err
			}
			if _, err := tx.RevealGameRoles(ctx, sqlc.RevealGameRolesParams{
				RevealedAt:    sql.NullTime{Time: now, Valid: true},
				GameSessionID: gameID,
			}); err != nil {
				return mapGameStorageError(err)
			}
		} else if decision.State.PhaseDeadlineAt != nil &&
			decision.State.PhaseKey != state.PhaseKey {
			if err := s.createTimer(
				ctx,
				tx,
				gameID,
				decision.State.PhaseKey,
				*decision.State.PhaseDeadlineAt,
				newVersion,
			); err != nil {
				return err
			}
		}

		resultJSON, _ := json.Marshal(decision.ResultSummary)
		if err := tx.CompleteGameIntent(ctx, sqlc.CompleteGameIntentParams{
			Status:            "executed",
			ResultCode:        sql.NullString{String: decision.ResultCode, Valid: decision.ResultCode != ""},
			ResultSummaryJson: rawMessagePointer(resultJSON),
			EventStartSeq:     sql.NullInt64{Int64: int64(session.CurrentEventSeq + 1), Valid: len(eventIDs) > 0},
			EventEndSeq:       sql.NullInt64{Int64: int64(seq), Valid: len(eventIDs) > 0},
			FinishedAt:        sql.NullTime{Time: now, Valid: true},
			ID:                intent.ID,
		}); err != nil {
			return mapGameStorageError(err)
		}
		result.IntentID = intent.ID
		result.Status = "executed"
		if result.ResultCode == "" {
			result.ResultCode = decision.ResultCode
		}
		if result.Summary == nil {
			result.Summary = decision.ResultSummary
		}
		result.StateVersion = newVersion
		result.EventSeq = seq
		result.EventRange = EventRange{
			StartSeq: session.CurrentEventSeq + 1,
			EndSeq:   seq,
		}
		return nil
	})
	if err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (s *Service) findOrCreateIntent(
	ctx context.Context,
	repo storage.Repository,
	session sqlc.GameSession,
	playerID string,
	actionType domain.ActionType,
	idempotencyKey string,
	payloadJSON []byte,
	payloadHash []byte,
	expectedVersion uint64,
) (sqlc.GameIntent, bool, error) {
	intent, err := repo.GetGameIntentByIdempotency(ctx, sqlc.GetGameIntentByIdempotencyParams{
		GameSessionID:  session.ID,
		IdempotencyKey: idempotencyKey,
	})
	if err == nil {
		if !equalBytes(intent.PayloadHash, payloadHash) {
			return sqlc.GameIntent{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The idempotency key was reused with a different request.",
				false,
				nil,
			)
		}
		if intent.Status != "executed" {
			return sqlc.GameIntent{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The previous game command is still in progress.",
				true,
				nil,
			)
		}
		return intent, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.GameIntent{}, false, mapGameStorageError(err)
	}

	intentID := s.ids.New()
	if err := repo.CreateGameIntent(ctx, sqlc.CreateGameIntentParams{
		ID:              intentID,
		GameSessionID:   session.ID,
		IdempotencyKey:  idempotencyKey,
		ActorType:       "player",
		ActorID:         playerID,
		IntentType:      string(actionType),
		PayloadJson:     rawMessagePointer(payloadJSON),
		PayloadHash:     payloadHash,
		ExpectedVersion: expectedVersion,
		TraceID:         "trace:" + intentID,
	}); err != nil {
		return sqlc.GameIntent{}, false, mapGameStorageError(err)
	}
	return sqlc.GameIntent{ID: intentID}, false, nil
}

func resultFromIntent(intent sqlc.GameIntent, currentVersion uint64) ActionResult {
	summary := map[string]any{}
	if intent.ResultSummaryJson != nil {
		_ = json.Unmarshal(*intent.ResultSummaryJson, &summary)
	}
	return ActionResult{
		IntentID:     intent.ID,
		Status:       intent.Status,
		ResultCode:   intent.ResultCode.String,
		Summary:      summary,
		StateVersion: currentVersion,
		EventSeq:     uint64(intent.EventEndSeq.Int64),
		EventRange: EventRange{
			StartSeq: uint64(intent.EventStartSeq.Int64),
			EndSeq:   uint64(intent.EventEndSeq.Int64),
		},
		Replay: true,
	}
}

func persistSearchClaim(
	ctx context.Context,
	repo storage.Repository,
	claimID string,
	gameID string,
	playerID string,
	intentID string,
	eventID string,
	decision domain.DecisionResult,
	now time.Time,
) error {
	pointID, _ := decision.ResultSummary["search_point_id"].(string)
	clueID, _ := decision.ResultSummary["clue_id"].(string)
	claimSlot := intValue(decision.ResultSummary["claim_slot"])
	if pointID == "" || clueID == "" || claimSlot <= 0 {
		return internalError()
	}
	if err := repo.CreateGameSearchClaim(ctx, sqlc.CreateGameSearchClaimParams{
		ID:               claimID,
		GameSessionID:    gameID,
		SearchPointID:    pointID,
		ClueID:           clueID,
		ClaimSlot:        uint32(claimSlot),
		ClaimantPlayerID: playerID,
		IntentID:         intentID,
		EventID:          eventID,
		ClaimedAt:        now,
	}); err != nil {
		return mapGameStorageError(err)
	}
	return nil
}

func (s *Service) createResult(
	ctx context.Context,
	repo storage.Repository,
	gameID string,
	stateVersion uint64,
	eventSeq uint64,
	end domain.EndResult,
	now time.Time,
) error {
	winnerJSON, _ := json.Marshal(map[string]any{"team": end.WinnerTeam})
	summaryJSON, _ := json.Marshal(end.Summary)
	hashInput := fmt.Sprintf("%s|%s|%d|%d|%s", gameID, end.ResultCode, stateVersion, eventSeq, string(summaryJSON))
	hash := sha256.Sum256([]byte(hashInput))
	if err := repo.CreateGameResult(ctx, sqlc.CreateGameResultParams{
		ID:                s.ids.New(),
		GameSessionID:     gameID,
		FinalStateVersion: stateVersion,
		FinalEventSeq:     eventSeq,
		ResultCode:        end.ResultCode,
		WinnerTeamJson:    winnerJSON,
		SummaryJson:       summaryJSON,
		ResultHash:        hash[:],
		SettledAt:         now,
	}); err != nil {
		return mapGameStorageError(err)
	}
	return nil
}

func mapDomainActionError(err error) error {
	message := err.Error()
	switch {
	case contains(message, "clue is already claimed"):
		return apperror.New("CLUE_ALREADY_CLAIMED", message, false, nil)
	case contains(message, "deadline"):
		return apperror.New("PHASE_NOT_ALLOWED", message, false, nil)
	case contains(message, "not allowed"), contains(message, "only valid"),
		contains(message, "quota"), contains(message, "voting rights"):
		return apperror.New("ACTION_NOT_ALLOWED", message, false, nil)
	case contains(message, "self voting"), contains(message, "target"):
		return apperror.New("TARGET_INVALID", message, false, nil)
	case contains(message, "host"):
		return hostRequiredError()
	default:
		return apperror.New("VALIDATION_ERROR", message, false, nil)
	}
}

func equalBytes(left []byte, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var result byte
	for index := range left {
		result |= left[index] ^ right[index]
	}
	return result == 0
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func contains(value string, fragment string) bool {
	return len(fragment) > 0 && len(value) >= len(fragment) &&
		stringContains(value, fragment)
}

func stringContains(value string, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func versionConflictError() error {
	return apperror.New(
		"STATE_VERSION_CONFLICT",
		"The game state changed. Refresh and retry.",
		true,
		nil,
	)
}
