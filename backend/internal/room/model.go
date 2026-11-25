package room

import (
	"encoding/json"
	"time"
)

type Capacity struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type RoomView struct {
	ID              string     `json:"id"`
	GameType        string     `json:"game_type"`
	ScriptVersionID string     `json:"script_version_id"`
	Title           string     `json:"title"`
	HostUserID      string     `json:"host_user_id"`
	Visibility      string     `json:"visibility"`
	HasPassword     bool       `json:"has_password"`
	Capacity        Capacity   `json:"capacity"`
	Status          string     `json:"status"`
	CurrentEventSeq uint64     `json:"room_event_seq"`
	ExpiresAt       *time.Time `json:"expires_at"`
	RowVersion      uint32     `json:"row_version"`
}

type MemberView struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	PublicNo     string     `json:"public_no"`
	Nickname     string     `json:"nickname"`
	AvatarURL    *string    `json:"avatar_url"`
	SeatNo       *int16     `json:"seat_no"`
	RoomRole     string     `json:"room_role"`
	MemberStatus string     `json:"member_status"`
	ReadyAt      *time.Time `json:"ready_at"`
	JoinedAt     time.Time  `json:"joined_at"`
}

type Projection struct {
	Room               RoomView     `json:"room"`
	Members            []MemberView `json:"members"`
	ViewerCapabilities []string     `json:"viewer_capabilities"`
	RoomEventSeq       uint64       `json:"room_event_seq"`
	CanStart           bool         `json:"can_start"`
}

type CreateInput struct {
	ScriptVersionID  string         `json:"script_version_id"`
	GameType         string         `json:"game_type"`
	RuleTemplateID   *string        `json:"rule_template_id"`
	Title            string         `json:"title"`
	Visibility       string         `json:"visibility"`
	Password         string         `json:"password"`
	Capacity         Capacity       `json:"capacity"`
	ModeConfig       map[string]any `json:"mode_config"`
	PrivacyConfirmed bool           `json:"privacy_confirmed"`
}

type CreateResult struct {
	Room             RoomView   `json:"room"`
	Member           MemberView `json:"member"`
	InviteHint       InviteHint `json:"invite_hint"`
	RoomEventSeq     uint64     `json:"room_event_seq"`
	IdempotentReplay bool       `json:"-"`
	InvitationToken  string     `json:"invitation_token,omitempty"`
}

type InviteHint struct {
	CodeEnabled bool `json:"code_enabled"`
}

type JoinInput struct {
	RoomCode        string `json:"room_code"`
	InvitationToken string `json:"invitation_token"`
	Password        string `json:"password"`
	PreferredSeatNo *int16 `json:"preferred_seat_no"`
}

type JoinResult struct {
	Member           MemberView `json:"member"`
	RoomProjection   RoomView   `json:"room_projection"`
	RoomEventSeq     uint64     `json:"room_event_seq"`
	GameID           *string    `json:"game_id"`
	IdempotentReplay bool       `json:"-"`
}

type LeaveInput struct {
	Reason               string `json:"reason"`
	ExpectedRoomEventSeq uint64 `json:"expected_room_event_seq"`
}

type LeaveResult struct {
	MemberStatus         string `json:"member_status"`
	RoomStatus           string `json:"room_status"`
	HostTransferRequired bool   `json:"host_transfer_required"`
}

type ReadyInput struct {
	Ready                bool   `json:"ready"`
	ExpectedRoomEventSeq uint64 `json:"expected_room_event_seq"`
}

type ReadyResult struct {
	Member       MemberView `json:"member"`
	ReadyCount   int64      `json:"ready_count"`
	CanStart     bool       `json:"can_start"`
	RoomEventSeq uint64     `json:"room_event_seq"`
}

type HostActionInput struct {
	Action               string `json:"action"`
	TargetMemberID       string `json:"target_member_id"`
	Reason               string `json:"reason"`
	ExpectedRoomEventSeq uint64 `json:"expected_room_event_seq"`
}

type HostActionResult struct {
	AcceptedAction       string       `json:"accepted_action"`
	RoomStatus           string       `json:"room_status"`
	Members              []MemberView `json:"members,omitempty"`
	RoomEventSeq         uint64       `json:"room_event_seq"`
	RequiresConfirmation bool         `json:"requires_confirmation"`
}

type MessageView struct {
	MessageID        string     `json:"message_id"`
	EventID          string     `json:"event_id"`
	Sender           SenderView `json:"sender"`
	Text             string     `json:"text"`
	ChatSeq          uint64     `json:"chat_seq"`
	SentAt           time.Time  `json:"sent_at"`
	ModerationStatus string     `json:"moderation_status"`
}

type SenderView struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type SendMessageInput struct {
	ClientMessageID string `json:"client_message_id"`
	Text            string `json:"text"`
}

type SendMessageResult struct {
	Message MessageView `json:"message"`
	EventID string      `json:"event_id"`
	ChatSeq uint64      `json:"chat_seq"`
	Replay  bool        `json:"-"`
}

type MessagePage struct {
	Items      []MessageView `json:"items"`
	NextCursor *uint64       `json:"next_cursor"`
}

type InvitationInput struct {
	MaxUses          uint32   `json:"max_uses"`
	ExpiresInSeconds int      `json:"expires_in_seconds"`
	InviteeUserIDs   []string `json:"invitee_user_ids"`
}

type InvitationResult struct {
	InvitationToken string    `json:"invitation_token"`
	ShareURL        string    `json:"share_url"`
	ExpiresAt       time.Time `json:"expires_at"`
	MaxUses         uint32    `json:"max_uses"`
}

type ResolveInvitationInput struct {
	InvitationToken string `json:"invitation_token"`
}

type ResolveInvitationResult struct {
	RoomID           string   `json:"room_id"`
	GameType         string   `json:"game_type"`
	ScriptTitle      string   `json:"script_title"`
	Capacity         Capacity `json:"capacity"`
	RequiresPassword bool     `json:"requires_password"`
	Joinable         bool     `json:"joinable"`
}

type PublicRoomCard struct {
	RoomID         string `json:"room_id"`
	GameType       string `json:"game_type"`
	ScriptTitle    string `json:"script_title"`
	HostPublicName string `json:"host_public_name"`
	Players        int64  `json:"players"`
	Capacity       int    `json:"capacity"`
	Visibility     string `json:"visibility"`
	HasPassword    bool   `json:"has_password"`
	Status         string `json:"status"`
}

type RoomEventView struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	Stream        string          `json:"stream"`
	StreamID      string          `json:"stream_id"`
	Seq           uint64          `json:"seq"`
	SchemaVersion int             `json:"schema_version"`
	ActorType     *string         `json:"actor_type"`
	ActorID       *string         `json:"actor_id"`
	Timestamp     time.Time       `json:"timestamp"`
	Payload       json.RawMessage `json:"payload"`
}
