package runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func (s *Service) Events(
	ctx context.Context,
	actorUserID string,
	gameID string,
	afterSeq uint64,
	limit int,
) (EventPage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return EventPage{}, mapGameStorageError(err)
	}
	player, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	})
	if err != nil {
		return EventPage{}, notGamePlayerError()
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return EventPage{}, err
	}
	role := roleForPlayer(state.Roles, player.ID)
	rows, err := s.repo.ListGameEventsAfter(ctx, sqlc.ListGameEventsAfterParams{
		GameSessionID: gameID,
		Seq:           afterSeq,
		Limit:         int32(limit + 1),
	})
	if err != nil {
		return EventPage{}, mapGameStorageError(err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	events := make([]EventView, 0, len(rows))
	for _, row := range rows {
		if !eventVisible(row, player.ID, role.ID) {
			continue
		}
		payload := map[string]any{}
		if row.PayloadJson != nil {
			if err := json.Unmarshal(*row.PayloadJson, &payload); err != nil {
				continue
			}
		} else if row.SecretPayloadID.Valid {
			privatePayload, err := s.decryptEventPayload(ctx, row)
			if err != nil {
				continue
			}
			payload = privatePayload
		}
		events = append(events, EventView{
			EventID:       row.ID,
			EventType:     row.EventType,
			Stream:        "game",
			StreamID:      gameID,
			Seq:           row.Seq,
			SchemaVersion: parseSchemaVersion(row.SchemaVersion),
			GameID:        gameID,
			Timestamp:     row.OccurredAt,
			Payload:       payload,
		})
	}
	_ = compiled
	return EventPage{
		Events:           events,
		CurrentSeq:       session.CurrentEventSeq,
		StateVersion:     session.StateVersion,
		HasMore:          hasMore,
		RequiresSnapshot: afterSeq+uint64(limit) < session.CurrentEventSeq,
	}, nil
}

func (s *Service) MyRole(
	ctx context.Context,
	actorUserID string,
	gameID string,
) (RoleView, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return RoleView{}, mapGameStorageError(err)
	}
	player, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	})
	if err != nil {
		return RoleView{}, notGamePlayerError()
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return RoleView{}, err
	}
	role := roleForPlayer(state.Roles, player.ID)
	if role.ID == "" {
		return RoleView{}, apperror.New("ROLE_NOT_BOUND", "The player has no active role.", false, nil)
	}
	character := characterByID(compiled.Config.Characters, role.CharacterID)
	privateScript := ""
	if character.PrivatePayloadID != "" {
		if value, err := s.decryptContentSecret(
			ctx,
			s.repo,
			character.PrivatePayloadID,
			"script_character:"+character.ID,
		); err == nil {
			privateScript = privateScriptBody(value)
		}
	}
	var goals []string
	_ = json.Unmarshal(character.PublicGoals, &goals)
	isHost, _ := s.roomPort.IsHost(ctx, s.repo, session.RoomID, actorUserID)
	actions := availableActions(state, compiled.Config, domainPlayerForID(state.Players, player.ID), role, isHost)
	used := state.SearchPointUses["quota:"+player.ID+":"+state.PhaseKey]
	return RoleView{
		PlayerID:         player.ID,
		RoleID:           role.ID,
		CharacterID:      role.CharacterID,
		DisplayName:      role.DisplayName,
		Faction:          role.FactionKey,
		Team:             role.TeamKey,
		PrivateScript:    privateScript,
		PublicProfile:    role.PublicProfile,
		Goals:            goals,
		SearchQuota:      SearchQuota{Used: used, Remaining: max(compiled.Config.Policy.DefaultSearchQuota-used, 0)},
		AvailableActions: actions,
	}, nil
}

func (s *Service) SearchPoints(
	ctx context.Context,
	actorUserID string,
	gameID string,
) (SearchPointsView, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return SearchPointsView{}, mapGameStorageError(err)
	}
	gamePlayer, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	})
	if err != nil {
		return SearchPointsView{}, notGamePlayerError()
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return SearchPointsView{}, err
	}
	phase := compiled.Config.Phases[state.PhaseIndex]
	if phase.Kind != domain.PhaseSearch || state.Status != domain.StatusRunning {
		return SearchPointsView{}, apperror.New(
			"PHASE_NOT_ALLOWED",
			"Search is not available in the current phase.",
			false,
			nil,
		)
	}
	used := state.SearchPointUses["quota:"+gamePlayer.ID+":"+phase.Key]
	remainingQuota := max(compiled.Config.Policy.DefaultSearchQuota-used, 0)
	views := make([]SearchPointView, 0)
	for _, point := range compiled.Config.SearchPoints {
		if point.PhaseKey != phase.Key {
			continue
		}
		uses := state.SearchPointUses[point.ID]
		remaining := point.MaxSearches - uses
		canSearch := remainingQuota > 0 && remaining > 0
		if point.ConflictPolicy == "repeatable" {
			canSearch = remainingQuota > 0
			remaining = -1
		}
		var remainingPtr *int
		if remaining >= 0 {
			value := remaining
			remainingPtr = &value
		}
		views = append(views, SearchPointView{
			PointID:             point.ID,
			Name:                point.Title,
			Description:         point.PublicDescription,
			State:               searchPointState(canSearch, remaining),
			RemainingGlobalUses: remainingPtr,
			CanSearch:           canSearch,
		})
	}
	return SearchPointsView{
		SearchPoints:            views,
		ViewerRemainingSearches: remainingQuota,
	}, nil
}

func (s *Service) Clues(
	ctx context.Context,
	actorUserID string,
	gameID string,
) ([]ClueView, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return nil, mapGameStorageError(err)
	}
	player, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	})
	if err != nil {
		return nil, notGamePlayerError()
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return nil, err
	}
	views := make([]ClueView, 0)
	for _, found := range state.FoundClues {
		if found.VisibilityScope != "public" && found.VisibilityScope != "room" &&
			found.ClaimantPlayerID != player.ID {
			continue
		}
		clue := clueByID(compiled.Config.Clues, found.ClueID)
		if clue.ID == "" {
			continue
		}
		body := clue.PublicText
		if clue.PrivatePayloadID != "" && found.ClaimantPlayerID == player.ID {
			if value, err := s.decryptContentSecret(
				ctx,
				s.repo,
				clue.PrivatePayloadID,
				"script_clue:"+clue.ID,
			); err == nil {
				if privateText := privateClueText(value); privateText != "" {
					body = privateText
				}
			}
		}
		views = append(views, ClueView{
			ClueID:        clue.ID,
			Title:         clue.Title,
			Body:          body,
			Visibility:    found.VisibilityScope,
			RevealedAt:    found.FoundAt,
			SourcePointID: found.SearchPointID,
		})
	}
	return views, nil
}

func (s *Service) Votes(
	ctx context.Context,
	actorUserID string,
	gameID string,
	roundKey string,
) (VotesView, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return VotesView{}, mapGameStorageError(err)
	}
	if _, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	}); err != nil {
		return VotesView{}, notGamePlayerError()
	}
	state, compiled, err := s.loadRuntime(ctx, s.repo, session)
	if err != nil {
		return VotesView{}, err
	}
	if roundKey == "" {
		roundKey = state.PhaseRoundKey
	}
	votes, err := s.repo.ListGameVotesForRound(ctx, sqlc.ListGameVotesForRoundParams{
		GameSessionID: gameID,
		RoundKey:      roundKey,
	})
	if err != nil {
		return VotesView{}, mapGameStorageError(err)
	}
	required := eligibleVoterCount(state, compiled.Config)
	revealed := session.Status == "completed" && compiled.Config.Policy.VoteVisibility == "after_close"
	views := make([]VoteView, 0)
	tally := make(map[string]int)
	for _, vote := range votes {
		target := ""
		if vote.TargetPlayerID.Valid {
			target = vote.TargetPlayerID.String
			if !vote.IsAbstain {
				tally[target]++
			}
		}
		if revealed || compiled.Config.Policy.VoteVisibility == "live" {
			views = append(views, VoteView{
				RoundKey:       vote.RoundKey,
				VoterPlayerID:  vote.VoterPlayerID,
				TargetPlayerID: target,
				Abstain:        vote.IsAbstain,
			})
		}
	}
	if !revealed && compiled.Config.Policy.VoteVisibility != "count_only" {
		tally = map[string]int{}
	}
	return VotesView{
		RoundKey:       roundKey,
		SubmittedCount: len(votes),
		RequiredCount:  required,
		Revealed:       revealed,
		Tally:          tally,
		Votes:          views,
	}, nil
}

func (s *Service) Result(
	ctx context.Context,
	actorUserID string,
	gameID string,
) (ResultView, error) {
	session, err := s.repo.GetGameSession(ctx, gameID)
	if err != nil {
		return ResultView{}, mapGameStorageError(err)
	}
	if _, err := s.repo.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
		GameSessionID: gameID,
		UserID:        sql.NullString{String: actorUserID, Valid: true},
	}); err != nil {
		return ResultView{}, notGamePlayerError()
	}
	result, err := s.repo.GetGameResultBySession(ctx, gameID)
	if err != nil {
		return ResultView{}, apperror.New("RESULT_NOT_FOUND", "The result is not ready.", false, nil)
	}
	var winner map[string]any
	var summary map[string]any
	_ = json.Unmarshal(result.WinnerTeamJson, &winner)
	_ = json.Unmarshal(result.SummaryJson, &summary)
	roleRows, err := s.repo.ListGameRoles(ctx, gameID)
	if err != nil {
		return ResultView{}, mapGameStorageError(err)
	}
	revealedRoles := make([]RevealedRole, 0, len(roleRows))
	for _, roleRow := range roleRows {
		role, err := s.decryptRole(ctx, s.repo, roleRow)
		if err != nil {
			return ResultView{}, err
		}
		revealedRoles = append(revealedRoles, RevealedRole{
			PlayerID:    role.PlayerID,
			RoleID:      role.ID,
			CharacterID: role.CharacterID,
			RoleName:    role.DisplayName,
			FactionKey:  role.FactionKey,
		})
	}
	_ = session
	return ResultView{
		ResultID:       result.ID,
		Status:         "completed",
		ResultCode:     result.ResultCode,
		WinnerTeam:     stringValue(winner["team"]),
		Summary:        summary,
		ResultHash:     fmt.Sprintf("%x", result.ResultHash),
		RevealedRoles:  revealedRoles,
		AIReviewStatus: "pending",
	}, nil
}

func (s *Service) decryptEventPayload(
	ctx context.Context,
	row sqlc.GameEvent,
) (map[string]any, error) {
	payload, err := s.repo.GetSecretPayload(ctx, row.SecretPayloadID.String)
	if err != nil {
		return nil, err
	}
	plaintext, err := s.cipher.Open(secret.Sealed{
		KeyID:       payload.KeyID,
		Nonce:       payload.Nonce,
		Ciphertext:  payload.Ciphertext,
		ContentHash: payload.ContentHash,
	}, "game:"+row.GameSessionID+":event:"+row.ID)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) decryptContentSecret(
	ctx context.Context,
	repo interface {
		GetSecretPayload(context.Context, string) (sqlc.SecretPayload, error)
	},
	secretID string,
	aad string,
) (map[string]any, error) {
	payload, err := repo.GetSecretPayload(ctx, secretID)
	if err != nil {
		return nil, err
	}
	plaintext, err := s.cipher.Open(secret.Sealed{
		KeyID:       payload.KeyID,
		Nonce:       payload.Nonce,
		Ciphertext:  payload.Ciphertext,
		ContentHash: payload.ContentHash,
	}, aad)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(plaintext, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func eventVisible(event sqlc.GameEvent, playerID string, roleID string) bool {
	switch event.VisibilityScope {
	case "public", "room":
		return true
	case "player":
		return visibilityContains(event.VisibilityJson, "players", playerID)
	case "role":
		return visibilityContains(event.VisibilityJson, "roles", roleID)
	case "role_group":
		return false
	default:
		return false
	}
}

func visibilityContains(raw *json.RawMessage, field string, value string) bool {
	if raw == nil {
		return false
	}
	var visibility map[string]any
	if err := json.Unmarshal(*raw, &visibility); err != nil {
		return false
	}
	items, _ := visibility[field].([]any)
	for _, item := range items {
		if text, ok := item.(string); ok && text == value {
			return true
		}
	}
	return false
}

func parseSchemaVersion(value string) int {
	if value == "" {
		return 1
	}
	first := value
	if index := strings.IndexByte(first, '.'); index >= 0 {
		first = first[:index]
	}
	var parsed int
	if _, err := fmt.Sscanf(first, "%d", &parsed); err != nil || parsed <= 0 {
		return 1
	}
	return parsed
}

func privateScriptBody(value map[string]any) string {
	for _, key := range []string{"private_script", "body", "text"} {
		if text, ok := value[key].(string); ok {
			return text
		}
	}
	return ""
}

func privateClueText(value map[string]any) string {
	for _, key := range []string{"text", "body", "private_text"} {
		if text, ok := value[key].(string); ok {
			return text
		}
	}
	return ""
}

func searchPointState(canSearch bool, remaining int) string {
	if canSearch {
		return "available"
	}
	if remaining == 0 {
		return "exhausted"
	}
	return "unavailable"
}

func characterByID(characters []domain.Character, id string) domain.Character {
	for _, character := range characters {
		if character.ID == id {
			return character
		}
	}
	return domain.Character{}
}

func clueByID(clues []domain.Clue, id string) domain.Clue {
	for _, clue := range clues {
		if clue.ID == id {
			return clue
		}
	}
	return domain.Clue{}
}

func eligibleVoterCount(state domain.RuntimeState, config domain.ScriptConfig) int {
	count := 0
	for _, player := range state.Players {
		if player.RuntimeStatus != "active" {
			continue
		}
		role := roleForPlayer(state.Roles, player.ID)
		if containsString(config.Policy.VotingCharacterKeys, role.CharacterKey) {
			count++
		}
	}
	return count
}
