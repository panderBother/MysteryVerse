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
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func (s *Service) Pause(
	ctx context.Context,
	actorUserID string,
	gameID string,
	input PauseInput,
	idempotencyKey string,
) (PauseResult, error) {
	if idempotencyKey == "" {
		return PauseResult{}, validationError("Idempotency-Key is required")
	}
	var result PauseResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		session, err := tx.GetGameSessionForUpdate(ctx, gameID)
		if err != nil {
			return mapGameStorageError(err)
		}
		if input.ExpectedVersion != session.StateVersion {
			return versionConflictError()
		}
		isHost, err := s.roomPort.IsHost(ctx, tx, session.RoomID, actorUserID)
		if err != nil || !isHost {
			return hostRequiredError()
		}
		player, err := tx.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
			GameSessionID: gameID,
			UserID:        sql.NullString{String: actorUserID, Valid: true},
		})
		if err != nil {
			return notGamePlayerError()
		}
		payload, _ := json.Marshal(input)
		hash := sha256.Sum256(payload)
		intent, replay, err := s.findOrCreateIntent(
			ctx,
			tx,
			session,
			player.ID,
			domain.ActionType("pause"),
			idempotencyKey,
			payload,
			hash[:],
			input.ExpectedVersion,
		)
		if err != nil {
			return err
		}
		if replay {
			summary := map[string]any{}
			if intent.ResultSummaryJson != nil {
				_ = json.Unmarshal(*intent.ResultSummaryJson, &summary)
			}
			result = PauseResult{
				GameID:       gameID,
				Status:       "paused",
				StateVersion: session.StateVersion,
				EventSeq:     session.CurrentEventSeq,
				RemainingMS:  int64(intValue(summary["remaining_ms"])),
				Replay:       true,
			}
			return nil
		}
		if session.Status == "paused" {
			if err := s.completeControlIntent(ctx, tx, intent.ID, "paused", 0, session); err != nil {
				return err
			}
			result = PauseResult{
				GameID:       gameID,
				Status:       "paused",
				StateVersion: session.StateVersion,
				EventSeq:     session.CurrentEventSeq,
				Replay:       true,
			}
			return nil
		}
		if session.Status != "running" {
			return apperror.New("PAUSE_NOT_ALLOWED", "The game cannot be paused now.", false, nil)
		}
		if _, err := tx.GetGameStateForUpdate(ctx, gameID); err != nil {
			return mapGameStorageError(err)
		}
		state, compiled, err := s.loadRuntime(ctx, tx, session)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		remaining := int64(0)
		if state.PhaseDeadlineAt != nil {
			remaining = max(state.PhaseDeadlineAt.Sub(now).Milliseconds(), 0)
		}
		state.Status = domain.StatusPaused
		state.PausedAt = &now
		state.RemainingMsAtPause = &remaining
		newVersion := session.StateVersion + 1
		seq := session.CurrentEventSeq + 1
		state.StateVersion = newVersion
		state.EventSeq = seq
		event := domain.DomainEvent{
			EventType:       "game.paused",
			EventCategory:   "incident",
			VisibilityScope: "room",
			Sensitivity:     0,
			Payload: map[string]any{
				"schema_version": "1.0.0",
				"remaining_ms":   remaining,
			},
			StateChanged: true,
		}
		eventID, err := s.persistDomainEvent(
			ctx,
			tx,
			gameID,
			seq,
			event,
			"player",
			player.ID,
			intent.ID,
			session.StateVersion,
			newVersion,
			now,
		)
		if err != nil {
			return err
		}
		_ = eventID
		if err := s.persistStateAndSession(ctx, tx, session, state, compiled.Config, newVersion, seq, "paused", false, now); err != nil {
			return err
		}
		if timer, err := tx.GetLatestGameTimerForUpdate(ctx, gameID); err == nil {
			if err := tx.UpdateGameTimer(ctx, sqlc.UpdateGameTimerParams{
				DeadlineAt:           timer.DeadlineAt,
				PausedAt:             sql.NullTime{Time: now, Valid: true},
				RemainingMsAtPause:   sql.NullInt64{Int64: remaining, Valid: true},
				Status:               "paused",
				ExpectedStateVersion: newVersion,
				ID:                   timer.ID,
			}); err != nil {
				return mapGameStorageError(err)
			}
		}
		result = PauseResult{
			GameID:       gameID,
			Status:       "paused",
			StateVersion: newVersion,
			EventSeq:     seq,
			RemainingMS:  remaining,
		}
		return s.completeControlIntent(ctx, tx, intent.ID, "paused", remaining, sqlc.GameSession{
			StateVersion:    newVersion,
			CurrentEventSeq: seq,
		})
	})
	if err != nil {
		return PauseResult{}, err
	}
	return result, nil
}

func (s *Service) Resume(
	ctx context.Context,
	actorUserID string,
	gameID string,
	input PauseInput,
	idempotencyKey string,
) (ResumeResult, error) {
	if idempotencyKey == "" {
		return ResumeResult{}, validationError("Idempotency-Key is required")
	}
	var result ResumeResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		session, err := tx.GetGameSessionForUpdate(ctx, gameID)
		if err != nil {
			return mapGameStorageError(err)
		}
		if input.ExpectedVersion != session.StateVersion {
			return versionConflictError()
		}
		isHost, err := s.roomPort.IsHost(ctx, tx, session.RoomID, actorUserID)
		if err != nil || !isHost {
			return hostRequiredError()
		}
		player, err := tx.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
			GameSessionID: gameID,
			UserID:        sql.NullString{String: actorUserID, Valid: true},
		})
		if err != nil {
			return notGamePlayerError()
		}
		payload, _ := json.Marshal(input)
		hash := sha256.Sum256(payload)
		intent, replay, err := s.findOrCreateIntent(
			ctx,
			tx,
			session,
			player.ID,
			domain.ActionType("resume"),
			idempotencyKey,
			payload,
			hash[:],
			input.ExpectedVersion,
		)
		if err != nil {
			return err
		}
		if replay {
			state, _, err := s.loadRuntime(ctx, tx, session)
			if err != nil {
				return err
			}
			result = ResumeResult{
				GameID:       gameID,
				Status:       string(state.Status),
				StateVersion: session.StateVersion,
				EventSeq:     session.CurrentEventSeq,
				DeadlineAt:   state.PhaseDeadlineAt,
				Replay:       true,
			}
			return nil
		}
		if session.Status == "running" {
			if err := s.completeControlIntent(ctx, tx, intent.ID, "running", 0, session); err != nil {
				return err
			}
			result = ResumeResult{
				GameID:       gameID,
				Status:       "running",
				StateVersion: session.StateVersion,
				EventSeq:     session.CurrentEventSeq,
				Replay:       true,
			}
			return nil
		}
		if session.Status != "paused" {
			return apperror.New("PAUSE_NOT_ALLOWED", "The game is not paused.", false, nil)
		}
		if _, err := tx.GetGameStateForUpdate(ctx, gameID); err != nil {
			return mapGameStorageError(err)
		}
		state, compiled, err := s.loadRuntime(ctx, tx, session)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		remaining := int64(0)
		if state.RemainingMsAtPause != nil {
			remaining = *state.RemainingMsAtPause
		}
		var deadline *time.Time
		if remaining > 0 {
			value := now.Add(time.Duration(remaining) * time.Millisecond)
			deadline = &value
		}
		state.Status = domain.StatusRunning
		state.PausedAt = nil
		state.RemainingMsAtPause = nil
		state.PhaseDeadlineAt = deadline
		newVersion := session.StateVersion + 1
		seq := session.CurrentEventSeq + 1
		state.StateVersion = newVersion
		state.EventSeq = seq
		event := domain.DomainEvent{
			EventType:       "game.resumed",
			EventCategory:   "incident",
			VisibilityScope: "room",
			Sensitivity:     0,
			Payload: map[string]any{
				"schema_version": "1.0.0",
				"deadline_at":    deadline,
			},
			StateChanged: true,
		}
		if _, err := s.persistDomainEvent(
			ctx,
			tx,
			gameID,
			seq,
			event,
			"player",
			player.ID,
			intent.ID,
			session.StateVersion,
			newVersion,
			now,
		); err != nil {
			return err
		}
		if err := s.persistStateAndSession(ctx, tx, session, state, compiled.Config, newVersion, seq, "running", false, now); err != nil {
			return err
		}
		if deadline != nil {
			if timer, err := tx.GetLatestGameTimerForUpdate(ctx, gameID); err == nil {
				if err := tx.UpdateGameTimer(ctx, sqlc.UpdateGameTimerParams{
					DeadlineAt:           *deadline,
					Status:               "scheduled",
					ExpectedStateVersion: newVersion,
					ID:                   timer.ID,
				}); err != nil {
					return mapGameStorageError(err)
				}
			} else if err := s.createTimer(ctx, tx, gameID, state.PhaseKey, *deadline, newVersion); err != nil {
				return err
			}
		}
		result = ResumeResult{
			GameID:       gameID,
			Status:       "running",
			StateVersion: newVersion,
			EventSeq:     seq,
			DeadlineAt:   deadline,
		}
		return s.completeControlIntent(ctx, tx, intent.ID, "running", remaining, sqlc.GameSession{
			StateVersion:    newVersion,
			CurrentEventSeq: seq,
		})
	})
	if err != nil {
		return ResumeResult{}, err
	}
	return result, nil
}

func (s *Service) PlayerExit(
	ctx context.Context,
	actorUserID string,
	gameID string,
) (LeaveGameResult, error) {
	var result LeaveGameResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		session, err := tx.GetGameSessionForUpdate(ctx, gameID)
		if err != nil {
			return mapGameStorageError(err)
		}
		if session.Status == "completed" || session.Status == "aborted" {
			return apperror.New("GAME_ALREADY_ENDED", "The game has already ended.", false, nil)
		}
		player, err := tx.GetGamePlayerByUser(ctx, sqlc.GetGamePlayerByUserParams{
			GameSessionID: gameID,
			UserID:        sql.NullString{String: actorUserID, Valid: true},
		})
		if err != nil {
			return notGamePlayerError()
		}
		if _, err := tx.GetGameStateForUpdate(ctx, gameID); err != nil {
			return mapGameStorageError(err)
		}
		state, compiled, err := s.loadRuntime(ctx, tx, session)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		for index := range state.Players {
			if state.Players[index].ID == player.ID {
				state.Players[index].RuntimeStatus = "left"
			}
		}
		if err := tx.UpdateGamePlayerRuntimeStatus(ctx, sqlc.UpdateGamePlayerRuntimeStatusParams{
			RuntimeStatus:   "left",
			LeftAt:          sql.NullTime{Time: now, Valid: true},
			ConnectionState: sql.NullString{String: "offline", Valid: true},
			GameSessionID:   gameID,
			ID:              player.ID,
		}); err != nil {
			return mapGameStorageError(err)
		}
		newVersion := session.StateVersion + 1
		seq := session.CurrentEventSeq + 1
		event := domain.DomainEvent{
			EventType:       "game.player_left",
			EventCategory:   "incident",
			VisibilityScope: "room",
			Sensitivity:     0,
			Payload: map[string]any{
				"schema_version": "1.0.0",
				"player_id":      player.ID,
			},
			StateChanged: true,
		}
		if _, err := s.persistDomainEvent(
			ctx,
			tx,
			gameID,
			seq,
			event,
			"player",
			player.ID,
			"",
			session.StateVersion,
			newVersion,
			now,
		); err != nil {
			return err
		}
		active := activePlayerCount(state)
		status := string(state.Status)
		if active == 0 {
			status = "aborted"
			state.Status = domain.StatusAborted
			state.ResultCode = "aborted_player_exit"
			seq++
			if _, err := s.persistDomainEvent(
				ctx,
				tx,
				gameID,
				seq,
				domain.DomainEvent{
					EventType:       "game.aborted",
					EventCategory:   "incident",
					VisibilityScope: "room",
					Sensitivity:     0,
					Payload: map[string]any{
						"schema_version": "1.0.0",
						"reason":         "all_players_left",
					},
					StateChanged: true,
				},
				"system",
				"",
				"",
				session.StateVersion,
				newVersion,
				now,
			); err != nil {
				return err
			}
		}
		state.StateVersion = newVersion
		state.EventSeq = seq
		if err := s.persistStateAndSession(
			ctx,
			tx,
			session,
			state,
			compiled.Config,
			newVersion,
			seq,
			status,
			active == 0,
			now,
		); err != nil {
			return err
		}
		if active == 0 {
			if err := s.createResult(ctx, tx, gameID, newVersion, seq, domain.EndResult{
				ResultCode: "aborted_player_exit",
				WinnerTeam: "none",
				Summary: map[string]any{
					"reason": "all_players_left",
				},
			}, now); err != nil {
				return err
			}
		}
		result = LeaveGameResult{
			PlayerID:     player.ID,
			Status:       status,
			StateVersion: newVersion,
			EventSeq:     seq,
		}
		return nil
	})
	if err != nil {
		return LeaveGameResult{}, err
	}
	return result, nil
}

func (s *Service) ExpireRunningGames(ctx context.Context) (int, error) {
	ids, err := s.repo.ListExpiredRunningGameIDs(ctx, sqlc.ListExpiredRunningGameIDsParams{
		PhaseDeadlineAt: sql.NullTime{Time: s.now().UTC(), Valid: true},
		Limit:           200,
	})
	if err != nil {
		return 0, mapGameStorageError(err)
	}
	processed := 0
	for _, gameID := range ids {
		if err := s.tickGame(ctx, gameID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Service) tickGame(ctx context.Context, gameID string) error {
	return s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		session, err := tx.GetGameSessionForUpdate(ctx, gameID)
		if err != nil {
			return mapGameStorageError(err)
		}
		if session.Status != "running" {
			return nil
		}
		if _, err := tx.GetGameStateForUpdate(ctx, gameID); err != nil {
			return mapGameStorageError(err)
		}
		state, compiled, err := s.loadRuntime(ctx, tx, session)
		if err != nil {
			return err
		}
		engine, err := domain.NewScriptMurderEngine(compiled.Config)
		if err != nil {
			return internalError()
		}
		now := s.now().UTC()
		decision, err := engine.Tick(state, compiled.Config, now)
		if err != nil {
			return mapDomainActionError(err)
		}
		if len(decision.Events) == 0 {
			return nil
		}
		if decision.State.Status == domain.StatusSettling {
			end, complete, err := engine.CheckEnd(decision.State, compiled.Config)
			if err != nil || !complete {
				return internalError()
			}
			decision.State.Status = domain.StatusCompleted
			decision.State.ResultCode = end.ResultCode
			decision.State.SettlementSummary = end.Summary
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
			decision.ResultCode = end.ResultCode
			decision.ResultSummary = end.Summary
		}
		newVersion := session.StateVersion + 1
		seq := session.CurrentEventSeq
		for _, event := range decision.Events {
			seq++
			if _, err := s.persistDomainEvent(
				ctx,
				tx,
				gameID,
				seq,
				event,
				"system",
				"",
				"",
				session.StateVersion,
				newVersion,
				now,
			); err != nil {
				return err
			}
		}
		decision.State.StateVersion = newVersion
		decision.State.EventSeq = seq
		status := string(decision.State.Status)
		ended := decision.State.Status == domain.StatusCompleted ||
			decision.State.Status == domain.StatusAborted
		if err := s.persistStateAndSession(
			ctx,
			tx,
			session,
			decision.State,
			compiled.Config,
			newVersion,
			seq,
			status,
			ended,
			now,
		); err != nil {
			return err
		}
		if decision.State.Status == domain.StatusCompleted {
			if err := s.createResult(ctx, tx, gameID, newVersion, seq, domain.EndResult{
				ResultCode: decision.ResultCode,
				WinnerTeam: stringValue(decision.ResultSummary["winner_team"]),
				Summary:    decision.ResultSummary,
			}, now); err != nil {
				return err
			}
			if _, err := tx.RevealGameRoles(ctx, sqlc.RevealGameRolesParams{
				RevealedAt:    sql.NullTime{Time: now, Valid: true},
				GameSessionID: gameID,
			}); err != nil {
				return mapGameStorageError(err)
			}
		} else if decision.State.Status == domain.StatusAborted {
			if err := s.createResult(ctx, tx, gameID, newVersion, seq, domain.EndResult{
				ResultCode: decision.State.ResultCode,
				WinnerTeam: "none",
				Summary:    decision.State.SettlementSummary,
			}, now); err != nil {
				return err
			}
		} else if decision.State.PhaseDeadlineAt != nil {
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
		return nil
	})
}

func (s *Service) persistStateAndSession(
	ctx context.Context,
	repo storage.Repository,
	session sqlc.GameSession,
	state domain.RuntimeState,
	config domain.ScriptConfig,
	newVersion uint64,
	seq uint64,
	status string,
	ended bool,
	now time.Time,
) error {
	stateHash := domain.StateHash(state)
	stateSecretID, err := s.storeSecret(ctx, repo, secretRecordInput{
		ScopeType:       "game",
		ScopeID:         session.ID,
		OwnerType:       "game_state",
		OwnerID:         session.ID,
		PurposeCode:     "runtime_secret",
		VersionNo:       uint32(newVersion),
		Sensitivity:     3,
		VisibilityScope: "runtime",
		Value:           state,
		AAD:             "game:" + session.ID + ":state",
	})
	if err != nil {
		return err
	}
	_ = config
	publicJSON, _ := json.Marshal(publicState(state, config))
	if err := repo.UpdateGameState(ctx, sqlc.UpdateGameStateParams{
		PublicStateJson:     publicJSON,
		SecretPayloadID:     stateSecretID,
		SchemaVersion:       config.SchemaVersion,
		PhaseKey:            state.PhaseKey,
		PhaseStartedAt:      nullableTimeParam(timePointer(state.PhaseStartedAt)),
		PhaseDeadlineAt:     nullableTimeParam(state.PhaseDeadlineAt),
		StateHash:           stateHash,
		LastEventSeq:        seq,
		MaterializerVersion: "murder-state-v1",
		GameSessionID:       session.ID,
	}); err != nil {
		return mapGameStorageError(err)
	}
	endedAt := sql.NullTime{}
	settleReason := sql.NullString{}
	if ended {
		endedAt = sql.NullTime{Time: now, Valid: true}
		settleReason = sql.NullString{String: state.ResultCode, Valid: state.ResultCode != ""}
	}
	if err := repo.UpdateGameSessionCommand(ctx, sqlc.UpdateGameSessionCommandParams{
		Status:          status,
		StateVersion:    newVersion,
		CurrentEventSeq: seq,
		StateHash:       stateHash,
		EndedAt:         endedAt,
		SettleReason:    settleReason,
		ID:              session.ID,
		StateVersion_2:  session.StateVersion,
	}); err != nil {
		if errors.Is(err, storage.ErrNoRowsAffected) {
			return versionConflictError()
		}
		return mapGameStorageError(err)
	}
	return nil
}

func (s *Service) completeControlIntent(
	ctx context.Context,
	repo storage.Repository,
	intentID string,
	status string,
	remainingMS int64,
	session sqlc.GameSession,
) error {
	summary, _ := json.Marshal(map[string]any{
		"status":       status,
		"remaining_ms": remainingMS,
	})
	return mapGameStorageError(repo.CompleteGameIntent(ctx, sqlc.CompleteGameIntentParams{
		Status:            "executed",
		ResultCode:        sql.NullString{String: status, Valid: true},
		ResultSummaryJson: rawMessagePointer(summary),
		EventStartSeq:     sql.NullInt64{Int64: int64(session.CurrentEventSeq), Valid: session.CurrentEventSeq > 0},
		EventEndSeq:       sql.NullInt64{Int64: int64(session.CurrentEventSeq), Valid: session.CurrentEventSeq > 0},
		FinishedAt:        sql.NullTime{Time: s.now().UTC(), Valid: true},
		ID:                intentID,
	}))
}

func activePlayerCount(state domain.RuntimeState) int {
	count := 0
	for _, player := range state.Players {
		if player.RuntimeStatus == "active" {
			count++
		}
	}
	return count
}
