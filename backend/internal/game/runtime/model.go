package runtime

import "time"

type StartInput struct {
	ExpectedRoomEventSeq uint64 `json:"expected_room_event_seq"`
	ContentVersionID     string `json:"content_version_id"`
	RulesAck             bool   `json:"rules_ack"`
}

type StartResult struct {
	GameID        string `json:"game_id"`
	RoomStatus    string `json:"room_status"`
	StateVersion  uint64 `json:"state_version"`
	GameEventSeq  uint64 `json:"game_event_seq"`
	EngineVersion string `json:"engine_version"`
	GameType      string `json:"game_type"`
	Replay        bool   `json:"-"`
}

type Viewer struct {
	PlayerID string `json:"player_id"`
	RoleID   string `json:"role_id"`
}

type PhaseView struct {
	Key        string     `json:"key"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	StartedAt  time.Time  `json:"started_at"`
	DeadlineAt *time.Time `json:"deadline_at"`
	Paused     bool       `json:"paused"`
}

type PlayerView struct {
	PlayerID        string `json:"player_id"`
	SeatNo          int    `json:"seat_no"`
	DisplayName     string `json:"display_name"`
	Status          string `json:"status"`
	ReadingComplete bool   `json:"reading_complete"`
}

type Projection struct {
	GameID             string         `json:"game_id"`
	GameType           string         `json:"game_type"`
	Status             string         `json:"status"`
	StateVersion       uint64         `json:"state_version"`
	EventSeq           uint64         `json:"event_seq"`
	Viewer             Viewer         `json:"viewer"`
	Phase              PhaseView      `json:"phase"`
	Players            []PlayerView   `json:"players"`
	PublicState        map[string]any `json:"public_state"`
	PrivateState       map[string]any `json:"private_state"`
	AvailableActions   []string       `json:"available_actions"`
	ViewerCapabilities []string       `json:"viewer_capabilities"`
}

type EventView struct {
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	Stream        string         `json:"stream"`
	StreamID      string         `json:"stream_id"`
	Seq           uint64         `json:"seq"`
	SchemaVersion int            `json:"schema_version"`
	GameID        string         `json:"game_id"`
	Timestamp     time.Time      `json:"timestamp"`
	Payload       map[string]any `json:"payload"`
}

type EventPage struct {
	Events           []EventView `json:"events"`
	CurrentSeq       uint64      `json:"current_seq"`
	StateVersion     uint64      `json:"state_version"`
	HasMore          bool        `json:"has_more"`
	RequiresSnapshot bool        `json:"requires_snapshot"`
}

type RoleView struct {
	PlayerID         string      `json:"player_id"`
	RoleID           string      `json:"role_id"`
	CharacterID      string      `json:"character_id"`
	DisplayName      string      `json:"display_name"`
	Faction          string      `json:"faction"`
	Team             string      `json:"team,omitempty"`
	PrivateScript    string      `json:"private_script"`
	PublicProfile    string      `json:"public_profile"`
	Goals            []string    `json:"goals"`
	SearchQuota      SearchQuota `json:"search_quota"`
	AvailableActions []string    `json:"available_actions"`
}

type SearchQuota struct {
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}

type ClueView struct {
	ClueID        string    `json:"clue_id"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	Visibility    string    `json:"visibility"`
	RevealedAt    time.Time `json:"revealed_at"`
	SourcePointID string    `json:"source_point_id"`
}

type SearchPointView struct {
	PointID             string `json:"point_id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	State               string `json:"state"`
	RemainingGlobalUses *int   `json:"remaining_global_uses"`
	CanSearch           bool   `json:"can_search"`
}

type SearchPointsView struct {
	SearchPoints            []SearchPointView `json:"search_points"`
	ViewerRemainingSearches int               `json:"viewer_remaining_searches"`
}

type SearchInput struct {
	SearchPointID   string `json:"search_point_id"`
	ExpectedVersion uint64 `json:"expected_version"`
	ClientRequestID string `json:"client_request_id"`
}

type VoteInput struct {
	TargetPlayerID  string `json:"target_player_id"`
	Abstain         bool   `json:"abstain"`
	ExpectedVersion uint64 `json:"expected_version"`
	ClientRequestID string `json:"client_request_id"`
	RoundKey        string `json:"round_key"`
}

type VoteView struct {
	RoundKey       string `json:"round_key"`
	VoterPlayerID  string `json:"voter_player_id"`
	TargetPlayerID string `json:"target_player_id,omitempty"`
	Abstain        bool   `json:"abstain"`
}

type VoteProgress struct {
	Submitted int `json:"submitted"`
	Required  int `json:"required"`
}

type VoteResult struct {
	Vote           VoteView     `json:"vote"`
	Progress       VoteProgress `json:"progress"`
	VoteVisibility string       `json:"vote_visibility"`
	StateVersion   uint64       `json:"state_version"`
	EventSeq       uint64       `json:"event_seq"`
}

type VotesView struct {
	RoundKey       string         `json:"round_key"`
	SubmittedCount int            `json:"submitted_count"`
	RequiredCount  int            `json:"required_count"`
	Revealed       bool           `json:"revealed"`
	Tally          map[string]int `json:"tally"`
	Votes          []VoteView     `json:"votes"`
}

type ActionResult struct {
	IntentID     string         `json:"intent_id"`
	Status       string         `json:"status"`
	ResultCode   string         `json:"result_code"`
	Summary      map[string]any `json:"summary,omitempty"`
	StateVersion uint64         `json:"state_version"`
	EventSeq     uint64         `json:"event_seq"`
	EventRange   EventRange     `json:"event_range"`
	Replay       bool           `json:"-"`
}

type EventRange struct {
	StartSeq uint64 `json:"start_seq"`
	EndSeq   uint64 `json:"end_seq"`
}

type PauseInput struct {
	ExpectedVersion uint64 `json:"expected_version"`
}

type PauseResult struct {
	GameID       string `json:"game_id"`
	Status       string `json:"status"`
	StateVersion uint64 `json:"state_version"`
	EventSeq     uint64 `json:"event_seq"`
	RemainingMS  int64  `json:"remaining_ms"`
	Replay       bool   `json:"-"`
}

type ResumeResult struct {
	GameID       string     `json:"game_id"`
	Status       string     `json:"status"`
	StateVersion uint64     `json:"state_version"`
	EventSeq     uint64     `json:"event_seq"`
	DeadlineAt   *time.Time `json:"deadline_at"`
	Replay       bool       `json:"-"`
}

type ResultView struct {
	ResultID       string         `json:"result_id"`
	Status         string         `json:"status"`
	ResultCode     string         `json:"result_code"`
	WinnerTeam     string         `json:"winner_team"`
	Summary        map[string]any `json:"summary"`
	ResultHash     string         `json:"result_hash"`
	RevealedRoles  []RevealedRole `json:"revealed_roles"`
	AIReviewStatus string         `json:"ai_review_status"`
}

type RevealedRole struct {
	PlayerID    string `json:"player_id"`
	RoleID      string `json:"role_id"`
	CharacterID string `json:"character_id"`
	RoleName    string `json:"role_name"`
	FactionKey  string `json:"faction_key"`
}

type LeaveGameResult struct {
	PlayerID     string `json:"player_id"`
	Status       string `json:"status"`
	StateVersion uint64 `json:"state_version"`
	EventSeq     uint64 `json:"event_seq"`
}
