package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Status string

const (
	StatusStarting  Status = "starting"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusSettling  Status = "settling"
	StatusCompleted Status = "completed"
	StatusAborted   Status = "aborted"
)

type PhaseKind string

const (
	PhaseRead           PhaseKind = "read"
	PhaseOpening        PhaseKind = "opening"
	PhaseSearch         PhaseKind = "search"
	PhasePrivateChat    PhaseKind = "private_chat"
	PhaseDiscussion     PhaseKind = "discussion"
	PhaseFinalStatement PhaseKind = "final_statement"
	PhaseVote           PhaseKind = "vote"
	PhaseSettlement     PhaseKind = "settlement"
	PhaseReplay         PhaseKind = "replay"
)

type ActionType string

const (
	ActionReadingComplete ActionType = "reading_complete"
	ActionSearch          ActionType = "search"
	ActionVote            ActionType = "vote"
	ActionAdvancePhase    ActionType = "advance_phase"
)

type Player struct {
	ID                 string         `json:"id"`
	UserID             string         `json:"user_id"`
	RoomMemberID       string         `json:"room_member_id"`
	SeatNo             int            `json:"seat_no"`
	DisplayName        string         `json:"display_name"`
	RuntimeStatus      string         `json:"runtime_status"`
	ReadingCompleted   bool           `json:"reading_completed"`
	SearchQuota        int            `json:"search_quota"`
	SearchQuotaByPhase map[string]int `json:"search_quota_by_phase"`
}

type Character struct {
	ID               string          `json:"id"`
	Key              string          `json:"key"`
	DisplayName      string          `json:"display_name"`
	RoleClass        string          `json:"role_class"`
	PublicProfile    string          `json:"public_profile"`
	PublicGoals      json.RawMessage `json:"public_goals"`
	Attributes       json.RawMessage `json:"attributes"`
	PrivatePayloadID string          `json:"private_payload_id,omitempty"`
}

type Role struct {
	ID               string `json:"id"`
	PlayerID         string `json:"player_id"`
	CharacterID      string `json:"character_id"`
	CharacterKey     string `json:"character_key"`
	DisplayName      string `json:"display_name"`
	FactionKey       string `json:"faction_key"`
	TeamKey          string `json:"team_key,omitempty"`
	PublicProfile    string `json:"public_profile"`
	RevealState      string `json:"reveal_state"`
	PrivatePayloadID string `json:"private_payload_id,omitempty"`
}

type Phase struct {
	ID             string       `json:"id"`
	Key            string       `json:"key"`
	Kind           PhaseKind    `json:"kind"`
	Sequence       int          `json:"sequence"`
	Title          string       `json:"title"`
	AllowedActions []ActionType `json:"allowed_actions"`
	TimeoutSeconds *int         `json:"timeout_seconds"`
	FallbackAction string       `json:"fallback_action"`
	IsTerminal     bool         `json:"is_terminal"`
}

type Clue struct {
	ID               string `json:"id"`
	Key              string `json:"key"`
	Title            string `json:"title"`
	ClueType         string `json:"clue_type"`
	PublicText       string `json:"public_text"`
	IsRepeatable     bool   `json:"is_repeatable"`
	MaxDiscoveries   *int   `json:"max_discoveries"`
	VisibilityScope  string `json:"visibility_scope"`
	Sensitivity      int    `json:"sensitivity"`
	PrivatePayloadID string `json:"private_payload_id,omitempty"`
}

type SearchPoint struct {
	ID                string `json:"id"`
	Key               string `json:"key"`
	PhaseKey          string `json:"phase_key"`
	Title             string `json:"title"`
	PublicDescription string `json:"public_description"`
	MaxSearches       int    `json:"max_searches"`
	ConflictPolicy    string `json:"conflict_policy"`
}

type SearchPointClue struct {
	SearchPointID string `json:"search_point_id"`
	ClueID        string `json:"clue_id"`
	Weight        int    `json:"weight"`
	IsDefault     bool   `json:"is_default"`
}

type MurderPolicy struct {
	SchemaVersion        string            `json:"schema_version"`
	DefaultSearchQuota   int               `json:"default_search_quota"`
	AllowAbstain         bool              `json:"allow_abstain"`
	AllowRevote          bool              `json:"allow_revote"`
	AllowSelfVote        bool              `json:"allow_self_vote"`
	VoteVisibility       string            `json:"vote_visibility"`
	TiePolicy            string            `json:"tie_policy"`
	CulpritCharacterKeys []string          `json:"culprit_character_keys"`
	VotingCharacterKeys  []string          `json:"voting_character_keys"`
	RoleFactions         map[string]string `json:"role_factions"`
	RoleTeams            map[string]string `json:"role_teams"`
}

type ScriptConfig struct {
	ScriptVersionID string            `json:"script_version_id"`
	Title           string            `json:"title"`
	EngineKey       string            `json:"engine_key"`
	EngineVersion   string            `json:"engine_version"`
	SchemaVersion   string            `json:"schema_version"`
	Policy          MurderPolicy      `json:"policy"`
	Phases          []Phase           `json:"phases"`
	Characters      []Character       `json:"characters"`
	Clues           []Clue            `json:"clues"`
	SearchPoints    []SearchPoint     `json:"search_points"`
	PointClues      []SearchPointClue `json:"point_clues"`
}

type FoundClue struct {
	ClueID           string    `json:"clue_id"`
	SearchPointID    string    `json:"search_point_id"`
	ClaimantPlayerID string    `json:"claimant_player_id"`
	ClaimSlot        int       `json:"claim_slot"`
	VisibilityScope  string    `json:"visibility_scope"`
	FoundAt          time.Time `json:"found_at"`
}

type Vote struct {
	RoundKey       string    `json:"round_key"`
	BallotType     string    `json:"ballot_type"`
	VoterPlayerID  string    `json:"voter_player_id"`
	TargetPlayerID string    `json:"target_player_id,omitempty"`
	Abstain        bool      `json:"abstain"`
	SubmittedAt    time.Time `json:"submitted_at"`
}

type RuntimeState struct {
	GameID             string         `json:"game_id"`
	GameType           string         `json:"game_type"`
	Status             Status         `json:"status"`
	StateVersion       uint64         `json:"state_version"`
	EventSeq           uint64         `json:"event_seq"`
	PhaseIndex         int            `json:"phase_index"`
	PhaseKey           string         `json:"phase_key"`
	PhaseStartedAt     time.Time      `json:"phase_started_at"`
	PhaseDeadlineAt    *time.Time     `json:"phase_deadline_at"`
	PausedAt           *time.Time     `json:"paused_at"`
	RemainingMsAtPause *int64         `json:"remaining_ms_at_pause"`
	Players            []Player       `json:"players"`
	Roles              []Role         `json:"roles"`
	FoundClues         []FoundClue    `json:"found_clues"`
	SearchPointUses    map[string]int `json:"search_point_uses"`
	Votes              []Vote         `json:"votes"`
	PhaseRoundKey      string         `json:"phase_round_key"`
	ResultCode         string         `json:"result_code,omitempty"`
	SettlementSummary  map[string]any `json:"settlement_summary,omitempty"`
}

type Actor struct {
	UserID   string
	PlayerID string
	RoleID   string
	IsHost   bool
}

type Action struct {
	Type    ActionType
	Payload map[string]any
}

type DomainEvent struct {
	EventType       string         `json:"event_type"`
	EventCategory   string         `json:"event_category"`
	VisibilityScope string         `json:"visibility_scope"`
	Visibility      map[string]any `json:"visibility,omitempty"`
	Sensitivity     int            `json:"sensitivity"`
	Payload         map[string]any `json:"payload"`
	StateChanged    bool           `json:"state_changed"`
}

type DecisionResult struct {
	State         RuntimeState   `json:"state"`
	Events        []DomainEvent  `json:"events"`
	ResultCode    string         `json:"result_code"`
	ResultSummary map[string]any `json:"result_summary,omitempty"`
}

type EndResult struct {
	Completed        bool           `json:"completed"`
	ResultCode       string         `json:"result_code"`
	WinnerTeam       string         `json:"winner_team"`
	CulpritPlayerIDs []string       `json:"culprit_player_ids"`
	AccusedPlayerIDs []string       `json:"accused_player_ids"`
	Summary          map[string]any `json:"summary"`
}

func (s Status) CanTransitionTo(target Status) bool {
	switch s {
	case StatusStarting:
		return target == StatusRunning || target == StatusAborted
	case StatusRunning:
		return target == StatusPaused || target == StatusSettling || target == StatusAborted
	case StatusPaused:
		return target == StatusRunning || target == StatusAborted
	case StatusSettling:
		return target == StatusCompleted || target == StatusAborted
	default:
		return false
	}
}

func (s Status) Terminal() bool {
	return s == StatusCompleted || s == StatusAborted
}

func (p PhaseKind) Validate() error {
	switch p {
	case PhaseRead, PhaseOpening, PhaseSearch, PhasePrivateChat, PhaseDiscussion,
		PhaseFinalStatement, PhaseVote, PhaseSettlement, PhaseReplay:
		return nil
	default:
		return fmt.Errorf("unsupported phase kind %q", p)
	}
}

func (p MurderPolicy) Validate() error {
	if p.SchemaVersion == "" {
		return errors.New("murder policy schema_version is required")
	}
	if p.DefaultSearchQuota < 1 {
		return errors.New("default_search_quota must be positive")
	}
	if p.VoteVisibility != "count_only" && p.VoteVisibility != "after_close" {
		return errors.New("vote_visibility must be count_only or after_close")
	}
	if p.TiePolicy != "no_elimination" {
		return fmt.Errorf("unsupported tie_policy %q", p.TiePolicy)
	}
	if len(p.CulpritCharacterKeys) == 0 {
		return errors.New("culprit_character_keys is required")
	}
	if len(p.VotingCharacterKeys) == 0 {
		return errors.New("voting_character_keys is required")
	}
	return nil
}
