package runtime

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func availableActions(
	state domain.RuntimeState,
	config domain.ScriptConfig,
	player domain.Player,
	role domain.Role,
	isHost bool,
) []string {
	if state.Status != domain.StatusRunning || player.RuntimeStatus != "active" {
		return []string{}
	}
	phase := config.Phases[state.PhaseIndex]
	actions := make([]string, 0)
	for _, action := range phase.AllowedActions {
		switch action {
		case domain.ActionReadingComplete:
			if !player.ReadingCompleted {
				actions = append(actions, string(action))
			}
		case domain.ActionSearch:
			key := "quota:" + player.ID + ":" + phase.Key
			if state.SearchPointUses[key] < config.Policy.DefaultSearchQuota {
				actions = append(actions, string(action))
			}
		case domain.ActionVote:
			if containsString(config.Policy.VotingCharacterKeys, role.CharacterKey) {
				actions = append(actions, string(action))
			}
		case domain.ActionAdvancePhase:
			if isHost {
				actions = append(actions, string(action))
			}
		}
	}
	return actions
}

func (s *Service) persistDomainEvent(
	ctx context.Context,
	repo storage.Repository,
	gameID string,
	seq uint64,
	event domain.DomainEvent,
	actorType string,
	actorID string,
	intentID string,
	stateVersionBefore uint64,
	stateVersionAfter uint64,
	now time.Time,
) (string, error) {
	payload, _ := json.Marshal(event.Payload)
	visibility, _ := json.Marshal(event.Visibility)
	before := sql.NullInt64{Int64: int64(stateVersionBefore), Valid: event.StateChanged}
	after := sql.NullInt64{Int64: int64(stateVersionAfter), Valid: event.StateChanged}
	eventID := s.ids.New()
	if err := repo.CreateGameEvent(ctx, sqlc.CreateGameEventParams{
		ID:                 eventID,
		GameSessionID:      gameID,
		Seq:                seq,
		EventType:          event.EventType,
		EventCategory:      event.EventCategory,
		ActorType:          sql.NullString{String: actorType, Valid: actorType != ""},
		ActorID:            sql.NullString{String: actorID, Valid: actorID != ""},
		IntentID:           sql.NullString{String: intentID, Valid: intentID != ""},
		VisibilityScope:    event.VisibilityScope,
		VisibilityJson:     rawMessagePointer(visibility),
		Sensitivity:        uint8(event.Sensitivity),
		PayloadJson:        rawMessagePointer(payload),
		PayloadHash:        hashBytes(payload),
		SchemaVersion:      "1.0.0",
		StateVersionBefore: before,
		StateVersionAfter:  after,
		OccurredAt:         now,
		RecordedAt:         now,
	}); err != nil {
		return "", mapGameStorageError(err)
	}
	return eventID, nil
}

func (s *Service) createTimer(
	ctx context.Context,
	repo storage.Repository,
	gameID string,
	phaseKey string,
	deadline time.Time,
	expectedVersion uint64,
) error {
	if err := repo.CreateGameTimer(ctx, sqlc.CreateGameTimerParams{
		ID:                   s.ids.New(),
		GameSessionID:        gameID,
		TimerKey:             phaseKey + ":" + s.ids.New(),
		PhaseKey:             phaseKey,
		DeadlineAt:           deadline,
		ExpectedStateVersion: expectedVersion,
	}); err != nil {
		return mapGameStorageError(err)
	}
	return nil
}

func mapGameStorageError(err error) error {
	if err == nil {
		return nil
	}
	var storageErr *database.Error
	if errors.As(err, &storageErr) {
		switch storageErr.Code {
		case database.CodeNotFound:
			return gameNotFoundError()
		case database.CodeConflict:
			return apperror.New(
				"STATE_VERSION_CONFLICT",
				"The game state changed. Refresh and retry.",
				true,
				nil,
			)
		case database.CodeDependencyUnavailable, database.CodeDeadlineExceeded:
			return apperror.New(
				"DEPENDENCY_ERROR",
				"The game service is temporarily unavailable.",
				true,
				nil,
			)
		}
	}
	return internalError()
}

func isStorageCode(err error, code string) bool {
	var storageErr *database.Error
	return errors.As(err, &storageErr) && storageErr.Code == code
}

func nullableStringParam(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableTimeParam(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func rawMessagePointer(value []byte) *json.RawMessage {
	raw := json.RawMessage(value)
	return &raw
}

func hashBytes(value []byte) []byte {
	sum := sha256.Sum256(value)
	return sum[:]
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func validationError(message string) error {
	return apperror.New("VALIDATION_ERROR", message, false, nil)
}

func internalError() error {
	return apperror.New("INTERNAL_ERROR", "The game operation failed.", true, nil)
}

func hostRequiredError() error {
	return apperror.New("HOST_REQUIRED", "The current user is not the room host.", false, nil)
}

func gameNotFoundError() error {
	return apperror.New("GAME_NOT_FOUND", "The game does not exist.", false, nil)
}

func notGamePlayerError() error {
	return apperror.New("NOT_GAME_PLAYER", "The user is not a game player.", false, nil)
}
