package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

type RoomStartMember struct {
	RoomMemberID string
	UserID       string
	SeatNo       int
	DisplayName  string
}

type RoomStartSnapshot struct {
	RoomID          string
	HostUserID      string
	ScriptVersionID string
	GameType        string
	RuleTemplateID  *string
	CapacityMin     int
	CapacityMax     int
	ModeConfig      json.RawMessage
	RoomEventSeq    uint64
	Members         []RoomStartMember
}

type RoomStartPort interface {
	LockForStart(
		ctx context.Context,
		repo storage.Repository,
		actorUserID string,
		roomID string,
		expectedRoomEventSeq uint64,
		now time.Time,
	) (RoomStartSnapshot, error)
	MarkStarted(
		ctx context.Context,
		repo storage.Repository,
		roomID string,
		gameID string,
		expectedRoomEventSeq uint64,
		now time.Time,
	) error
	IsHost(
		ctx context.Context,
		repo storage.Repository,
		roomID string,
		userID string,
	) (bool, error)
}
