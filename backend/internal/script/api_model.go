package script

import (
	"encoding/json"
	"time"
)

type TargetPlayers struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type CreateScriptInput struct {
	Title           string        `json:"title"`
	GameType        string        `json:"game_type"`
	RightsType      string        `json:"rights_type"`
	Language        string        `json:"language"`
	TargetPlayers   TargetPlayers `json:"target_players"`
	DurationMinutes int           `json:"duration_minutes"`
	Difficulty      string        `json:"difficulty"`
	ContentRating   string        `json:"content_rating"`
	RuleTemplateID  *string       `json:"rule_template_id"`
}

type CreateScriptResult struct {
	Script           Script  `json:"script"`
	Version          Version `json:"version"`
	IdempotentReplay bool    `json:"-"`
}

type UpdateScriptPatch struct {
	Title           *string `json:"title"`
	Summary         *string `json:"summary"`
	CoverAssetURI   *string `json:"cover_asset_uri"`
	Language        *string `json:"language"`
	ContentRating   *string `json:"content_rating"`
	MinPlayers      *int    `json:"min_players"`
	MaxPlayers      *int    `json:"max_players"`
	DurationMinutes *int    `json:"duration_minutes"`
	Difficulty      *string `json:"difficulty"`
}

type UpdateScriptInput struct {
	RowVersion uint32            `json:"row_version"`
	Patch      UpdateScriptPatch `json:"patch"`
}

type DeleteScriptInput struct {
	RowVersion uint32 `json:"row_version"`
	Reason     string `json:"reason"`
}

type DeleteScriptResult struct {
	ScriptID   string `json:"script_id"`
	Status     string `json:"status"`
	RowVersion uint32 `json:"row_version"`
}

type CreateDraftInput struct {
	DraftName      string  `json:"draft_name"`
	RuleTemplateID *string `json:"rule_template_id"`
}

type ValidateInput struct {
	ExpectedRowVersion uint32 `json:"expected_row_version"`
	Profile            string `json:"validation_profile"`
}

type PublishInput struct {
	Channel            string `json:"channel"`
	ExpectedRowVersion uint32 `json:"expected_row_version"`
	ContentRatingAck   bool   `json:"content_rating_confirm"`
}

type PublishResult struct {
	Publication Publication `json:"publication"`
	Version     Version     `json:"version"`
	Replay      bool        `json:"-"`
}

type UnpublishInput struct {
	ReasonCode     string `json:"reason_code"`
	Reason         string `json:"reason"`
	ExpectedStatus string `json:"expected_status"`
}

type UnpublishResult struct {
	Publication Publication `json:"publication"`
	Version     Version     `json:"version"`
	Replay      bool        `json:"-"`
}

type ObjectMutationInput struct {
	ExpectedRowVersion uint32          `json:"expected_row_version"`
	Object             json.RawMessage `json:"object"`
}

type ObjectMutationResult struct {
	ObjectType        string `json:"object_type"`
	ObjectID          string `json:"object_id"`
	Object            any    `json:"object"`
	VersionRowVersion uint32 `json:"version_row_version"`
}

type ChapterInput struct {
	ChapterKey string          `json:"chapter_key"`
	Title      string          `json:"title"`
	SequenceNo int             `json:"sequence_no"`
	Summary    *string         `json:"summary"`
	SourceRefs json.RawMessage `json:"source_refs"`
}

type StageInput struct {
	ChapterID       *string          `json:"chapter_id"`
	StageKey        string           `json:"stage_key"`
	PhaseType       string           `json:"phase_type"`
	SequenceNo      int              `json:"sequence_no"`
	Title           string           `json:"title"`
	EntryConditions *json.RawMessage `json:"entry_conditions"`
	ExitConditions  *json.RawMessage `json:"exit_conditions"`
	TimeoutSeconds  *int             `json:"timeout_seconds"`
	FallbackAction  string           `json:"fallback_action"`
	AllowedActions  []string         `json:"allowed_actions"`
	Visibility      string           `json:"visibility"`
	IsTerminal      bool             `json:"is_terminal"`
}

type CharacterInput struct {
	CharacterKey      string          `json:"character_key"`
	DisplayName       string          `json:"display_name"`
	RoleClass         string          `json:"role_class"`
	PublicProfile     string          `json:"public_profile"`
	PublicGoals       []string        `json:"public_goals"`
	Attributes        json.RawMessage `json:"attributes"`
	PrivateProfileRef *string         `json:"private_profile_ref"`
	Visibility        string          `json:"visibility"`
	Sensitivity       string          `json:"sensitivity"`
	SourceRefs        json.RawMessage `json:"source_refs"`
	GeneratedOrigin   string          `json:"generated_origin"`
	LockedFields      []string        `json:"locked_fields"`
	SortOrder         int             `json:"sort_order"`
}

type ClueInput struct {
	ClueKey        string          `json:"clue_key"`
	Title          string          `json:"title"`
	PublicText     *string         `json:"public_text"`
	PrivateTextRef *string         `json:"private_text_ref"`
	ClueType       string          `json:"clue_type"`
	IsRepeatable   bool            `json:"is_repeatable"`
	MaxDiscoveries *int            `json:"max_discoveries"`
	Visibility     string          `json:"visibility"`
	Sensitivity    string          `json:"sensitivity"`
	RevealGateID   *string         `json:"reveal_gate_id"`
	RelatedFactIDs []string        `json:"related_fact_ids"`
	SourceRefs     json.RawMessage `json:"source_refs"`
	SortOrder      int             `json:"sort_order"`
}

type EndingInput struct {
	EndingKey   string   `json:"ending_key"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Conditions  []string `json:"conditions"`
	IsHidden    bool     `json:"is_hidden"`
	Visibility  string   `json:"visibility"`
	SortOrder   int      `json:"sort_order"`
}

type AssetInput struct {
	AssetKey    string          `json:"asset_key"`
	AssetType   string          `json:"asset_type"`
	URI         string          `json:"uri"`
	MIMEType    *string         `json:"mime_type"`
	SHA256      *string         `json:"sha256"`
	ByteSize    *int64          `json:"byte_size"`
	Visibility  string          `json:"visibility"`
	Sensitivity string          `json:"sensitivity"`
	Metadata    json.RawMessage `json:"metadata"`
	SortOrder   int             `json:"sort_order"`
}

type SearchPointInput struct {
	PointKey          string          `json:"point_key"`
	StageID           string          `json:"stage_id"`
	SceneID           *string         `json:"scene_id"`
	Title             string          `json:"title"`
	PublicDescription string          `json:"public_description"`
	MaxSearches       int             `json:"max_searches"`
	ConflictPolicy    string          `json:"conflict_policy"`
	Visibility        string          `json:"visibility"`
	RevealGateID      *string         `json:"reveal_gate_id"`
	Metadata          json.RawMessage `json:"metadata"`
}

type SearchPointClueInput struct {
	SearchPointID string          `json:"search_point_id"`
	ClueID        string          `json:"clue_id"`
	Weight        int             `json:"weight"`
	IsDefault     bool            `json:"is_default"`
	Metadata      json.RawMessage `json:"metadata"`
}

type ScriptDetail struct {
	Script  Script  `json:"script"`
	Version Version `json:"version"`
}

type PublicScriptView struct {
	ScriptID        string     `json:"script_id"`
	OwnerUserID     string     `json:"owner_user_id"`
	Title           string     `json:"title"`
	Summary         *string    `json:"summary"`
	GameType        string     `json:"game_type"`
	CoverAssetURI   *string    `json:"cover_asset_uri"`
	Language        string     `json:"language"`
	ContentRating   string     `json:"content_rating"`
	MinPlayers      int        `json:"min_players"`
	MaxPlayers      int        `json:"max_players"`
	DurationMinutes int        `json:"duration_minutes"`
	Difficulty      string     `json:"difficulty"`
	PublishedAt     *time.Time `json:"published_at"`
}
