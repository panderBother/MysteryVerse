package room

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/ports"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type StartCoordinator struct {
	ids id.Generator
}

func NewStartCoordinator() *StartCoordinator {
	return &StartCoordinator{ids: id.ULIDGenerator{}}
}

func (c *StartCoordinator) LockForStart(
	ctx context.Context,
	repo storage.Repository,
	actorUserID string,
	roomID string,
	expectedRoomEventSeq uint64,
	now time.Time,
) (ports.RoomStartSnapshot, error) {
	roomRecord, err := repo.GetGameRoomForUpdate(ctx, roomID)
	if err != nil {
		return ports.RoomStartSnapshot{}, mapRoomStorageError(err)
	}
	if roomRecord.HostUserID != actorUserID {
		return ports.RoomStartSnapshot{}, hostRequiredError()
	}
	if roomRecord.Status != "waiting" {
		if isClosedStatus(roomRecord.Status) {
			return ports.RoomStartSnapshot{}, roomClosedError()
		}
		return ports.RoomStartSnapshot{}, roomNotWaitingError()
	}
	if isExpired(roomRecord, now) {
		return ports.RoomStartSnapshot{}, roomClosedError()
	}
	if expectedRoomEventSeq != roomRecord.CurrentEventSeq {
		return ports.RoomStartSnapshot{}, versionConflictError()
	}

	members, err := repo.ListActiveRoomMembers(ctx, roomID)
	if err != nil {
		return ports.RoomStartSnapshot{}, mapRoomStorageError(err)
	}
	if len(members) < int(roomRecord.CapacityMin) ||
		len(members) > int(roomRecord.CapacityMax) {
		return ports.RoomStartSnapshot{}, apperror.New(
			"PLAYER_COUNT_MISMATCH",
			"The room player count does not satisfy the content configuration.",
			false,
			nil,
		)
	}
	snapshotMembers := make([]ports.RoomStartMember, 0, len(members))
	for _, member := range members {
		if member.MemberStatus != "ready" {
			return ports.RoomStartSnapshot{}, apperror.New(
				"PLAYERS_NOT_READY",
				"All players must be ready before the game can start.",
				false,
				map[string]any{"member_id": member.ID},
			)
		}
		if !member.SeatNo.Valid || !member.UserID.Valid {
			return ports.RoomStartSnapshot{}, apperror.New(
				"PLAYER_COUNT_MISMATCH",
				"The room contains an invalid player seat.",
				false,
				nil,
			)
		}
		snapshotMembers = append(snapshotMembers, ports.RoomStartMember{
			RoomMemberID: member.ID,
			UserID:       member.UserID.String,
			SeatNo:       int(member.SeatNo.Int16),
			DisplayName:  member.Nickname,
		})
	}

	return ports.RoomStartSnapshot{
		RoomID:          roomRecord.ID,
		HostUserID:      roomRecord.HostUserID,
		ScriptVersionID: roomRecord.ScriptVersionID,
		GameType:        roomRecord.GameType,
		RuleTemplateID:  nullableString(roomRecord.RuleTemplateID),
		CapacityMin:     int(roomRecord.CapacityMin),
		CapacityMax:     int(roomRecord.CapacityMax),
		ModeConfig:      append(json.RawMessage(nil), roomRecord.ModeConfigJson...),
		RoomEventSeq:    roomRecord.CurrentEventSeq,
		Members:         snapshotMembers,
	}, nil
}

func (c *StartCoordinator) MarkStarted(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
	gameID string,
	expectedRoomEventSeq uint64,
	now time.Time,
) error {
	roomRecord, err := repo.GetGameRoomForUpdate(ctx, roomID)
	if err != nil {
		return mapRoomStorageError(err)
	}
	if roomRecord.Status != "waiting" {
		return roomNotWaitingError()
	}
	if roomRecord.CurrentEventSeq != expectedRoomEventSeq {
		return versionConflictError()
	}
	seq := roomRecord.CurrentEventSeq + 1
	payload, _ := json.Marshal(map[string]any{
		"schema_version": "1.0.0",
		"game_id":        gameID,
	})
	if err := repo.CreateRoomEvent(ctx, sqlc.CreateRoomEventParams{
		ID:              c.ids.New(),
		RoomID:          roomID,
		Seq:             seq,
		EventType:       "room.game_started",
		ActorType:       sql.NullString{String: "user", Valid: true},
		ActorID:         sql.NullString{String: roomRecord.HostUserID, Valid: true},
		VisibilityScope: "room",
		PayloadJson:     payload,
		PayloadHash:     hashBytes(payload),
		OccurredAt:      now,
	}); err != nil {
		return mapRoomStorageError(err)
	}
	if err := repo.UpdateGameRoomMutation(ctx, sqlc.UpdateGameRoomMutationParams{
		Status:          "in_game",
		CurrentEventSeq: seq,
		StartedAt:       sql.NullTime{Time: now, Valid: true},
		ClosedAt:        roomRecord.ClosedAt,
		ID:              roomID,
	}); err != nil {
		return mapRoomStorageError(err)
	}
	return nil
}

func (c *StartCoordinator) IsHost(
	ctx context.Context,
	repo storage.Repository,
	roomID string,
	userID string,
) (bool, error) {
	roomRecord, err := repo.GetGameRoom(ctx, roomID)
	if err != nil {
		return false, mapRoomStorageError(err)
	}
	return roomRecord.HostUserID == userID, nil
}
