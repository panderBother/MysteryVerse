package script

import (
	"encoding/json"
	"time"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type ValidationState string

const (
	ValidationStateChecked     ValidationState = "checked"
	ValidationStateUnsupported ValidationState = "unsupported"
	ValidationStatePending     ValidationState = "pending"
)

type ValidationIssue struct {
	ErrorCode string          `json:"error_code"`
	Severity  Severity        `json:"severity"`
	Path      string          `json:"path"`
	Message   string          `json:"message"`
	State     ValidationState `json:"state"`
	ObjectID  *string         `json:"object_id,omitempty"`
}

type ValidationReport struct {
	ScriptID          string            `json:"script_id"`
	ScriptVersionID   string            `json:"script_version_id"`
	VersionRowVersion uint32            `json:"version_row_version"`
	Status            string            `json:"status"`
	BlockingCount     int               `json:"blocking_issue_count"`
	WarningCount      int               `json:"warning_issue_count"`
	InfoCount         int               `json:"info_issue_count"`
	Issues            []ValidationIssue `json:"issues"`
	ValidatedAt       time.Time         `json:"validated_at"`
}

func (r ValidationReport) HasBlockingIssues() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

type Script struct {
	ID                 string     `json:"script_id"`
	OwnerUserID        string     `json:"owner_user_id"`
	PublicSlug         *string    `json:"public_slug,omitempty"`
	Title              string     `json:"title"`
	Summary            *string    `json:"summary"`
	GameType           string     `json:"game_type"`
	SourceKind         string     `json:"source_kind"`
	RightsType         string     `json:"rights_type"`
	Status             string     `json:"status"`
	CoverAssetURI      *string    `json:"cover_asset_uri"`
	Language           string     `json:"language"`
	ContentRating      string     `json:"content_rating"`
	MinPlayers         int        `json:"min_players"`
	MaxPlayers         int        `json:"max_players"`
	DurationMinutes    int        `json:"duration_minutes"`
	Difficulty         string     `json:"difficulty"`
	AIAssistedRatioBP  int        `json:"ai_assisted_ratio_bp"`
	ActiveDraftVersion *string    `json:"active_draft_version_id"`
	PublishedVersion   *string    `json:"published_version_id"`
	RowVersion         uint32     `json:"row_version"`
	DeletedAt          *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Version struct {
	ID                 string          `json:"version_id"`
	ScriptID           string          `json:"script_id"`
	VersionNo          int             `json:"version_no"`
	VersionKind        string          `json:"version_kind"`
	Status             string          `json:"status"`
	SchemaVersion      string          `json:"schema_version"`
	WorkflowVersion    string          `json:"workflow_version"`
	BaseVersionID      *string         `json:"base_version_id"`
	RuleTemplateID     *string         `json:"rule_template_id"`
	Blueprint          json.RawMessage `json:"blueprint"`
	PackageURI         *string         `json:"package_uri"`
	PackageHash        []byte          `json:"-"`
	Manifest           json.RawMessage `json:"manifest,omitempty"`
	ValidationReport   json.RawMessage `json:"validation_report,omitempty"`
	ValidationStatus   string          `json:"validation_status"`
	BlockingIssueCount int             `json:"blocking_issue_count"`
	CreatedBy          string          `json:"created_by"`
	FrozenAt           *time.Time      `json:"frozen_at"`
	PublishedAt        *time.Time      `json:"published_at"`
	RowVersion         uint32          `json:"row_version"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type Chapter struct {
	ID         string          `json:"chapter_id"`
	ChapterKey string          `json:"chapter_key"`
	Title      string          `json:"title"`
	SequenceNo int             `json:"sequence_no"`
	Summary    *string         `json:"summary"`
	SourceRefs json.RawMessage `json:"source_refs"`
	RowVersion uint32          `json:"row_version"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type Stage struct {
	ID              string           `json:"stage_id"`
	ChapterID       *string          `json:"chapter_id"`
	Key             string           `json:"stage_key"`
	Type            string           `json:"phase_type"`
	SequenceNo      int              `json:"sequence_no"`
	Title           string           `json:"title"`
	EntryConditions *json.RawMessage `json:"entry_conditions,omitempty"`
	ExitConditions  *json.RawMessage `json:"exit_conditions,omitempty"`
	TimeoutSeconds  *int             `json:"timeout_seconds"`
	FallbackAction  string           `json:"fallback_action"`
	AllowedActions  []string         `json:"allowed_actions"`
	Visibility      string           `json:"visibility"`
	IsTerminal      bool             `json:"is_terminal"`
	RowVersion      uint32           `json:"row_version"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type Character struct {
	ID                string          `json:"character_id"`
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
	RowVersion        uint32          `json:"row_version"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	PrivatePayloadID  *string         `json:"-"`
}

type Clue struct {
	ID               string          `json:"clue_id"`
	ClueKey          string          `json:"clue_key"`
	Title            string          `json:"title"`
	PublicText       *string         `json:"public_text"`
	PrivateTextRef   *string         `json:"private_text_ref"`
	ClueType         string          `json:"clue_type"`
	IsRepeatable     bool            `json:"is_repeatable"`
	MaxDiscoveries   *int            `json:"max_discoveries"`
	Visibility       string          `json:"visibility"`
	Sensitivity      string          `json:"sensitivity"`
	RevealGateID     *string         `json:"reveal_gate_id"`
	RelatedFactIDs   []string        `json:"related_fact_ids"`
	SourceRefs       json.RawMessage `json:"source_refs"`
	SortOrder        int             `json:"sort_order"`
	RowVersion       uint32          `json:"row_version"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	PrivatePayloadID *string         `json:"-"`
}

type Ending struct {
	ID          string          `json:"ending_id"`
	EndingKey   string          `json:"ending_key"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Conditions  json.RawMessage `json:"conditions"`
	IsHidden    bool            `json:"is_hidden"`
	Visibility  string          `json:"visibility"`
	SortOrder   int             `json:"sort_order"`
	RowVersion  uint32          `json:"row_version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Asset struct {
	ID          string          `json:"asset_id"`
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
	RowVersion  uint32          `json:"row_version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type SearchPoint struct {
	ID                string          `json:"search_point_id"`
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
	RowVersion        uint32          `json:"row_version"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type SearchPointClue struct {
	SearchPointID string          `json:"search_point_id"`
	ClueID        string          `json:"clue_id"`
	Weight        int             `json:"weight"`
	IsDefault     bool            `json:"is_default"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}

type Content struct {
	Script       Script            `json:"script"`
	Version      Version           `json:"version"`
	Chapters     []Chapter         `json:"chapters"`
	Stages       []Stage           `json:"stages"`
	Characters   []Character       `json:"characters"`
	Clues        []Clue            `json:"clues"`
	Endings      []Ending          `json:"endings"`
	Assets       []Asset           `json:"assets"`
	SearchPoints []SearchPoint     `json:"search_points"`
	PointClues   []SearchPointClue `json:"point_clues"`
}

type PlayableVersion struct {
	Script       Script
	Version      Version
	RuleTemplate *RuleTemplate
	Content      Content
}

type RuleTemplate struct {
	ID            string          `json:"id"`
	TemplateKey   string          `json:"template_key"`
	VersionNo     int             `json:"version_no"`
	GameType      string          `json:"game_type"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	SchemaVersion string          `json:"schema_version"`
	EngineVersion string          `json:"engine_version"`
	Config        json.RawMessage `json:"config"`
	ContentHash   []byte          `json:"-"`
}

type ScriptCatalogItem struct {
	ScriptID        string    `json:"script_id"`
	OwnerUserID     string    `json:"owner_user_id"`
	Title           string    `json:"title"`
	Summary         *string   `json:"summary"`
	GameType        string    `json:"game_type"`
	CoverAssetURI   *string   `json:"cover_asset_uri"`
	Language        string    `json:"language"`
	ContentRating   string    `json:"content_rating"`
	MinPlayers      int       `json:"min_players"`
	MaxPlayers      int       `json:"max_players"`
	DurationMinutes int       `json:"duration_minutes"`
	Difficulty      string    `json:"difficulty"`
	ScriptVersionID string    `json:"script_version_id"`
	VersionNo       int       `json:"version_no"`
	SchemaVersion   string    `json:"schema_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CatalogPage struct {
	Items      []ScriptCatalogItem `json:"items"`
	NextCursor *string             `json:"next_cursor"`
}

type VersionSummary struct {
	VersionID          string     `json:"version_id"`
	ScriptID           string     `json:"script_id"`
	VersionNo          int        `json:"version_no"`
	VersionKind        string     `json:"version_kind"`
	Status             string     `json:"status"`
	SchemaVersion      string     `json:"schema_version"`
	ValidationStatus   string     `json:"validation_status"`
	BlockingIssueCount int        `json:"blocking_issue_count"`
	CreatedAt          time.Time  `json:"created_at"`
	FrozenAt           *time.Time `json:"frozen_at"`
	PublishedAt        *time.Time `json:"published_at"`
	RowVersion         uint32     `json:"row_version"`
}

type Publication struct {
	ID            string     `json:"publication_id"`
	ScriptID      string     `json:"script_id"`
	VersionID     string     `json:"script_version_id"`
	Channel       string     `json:"channel"`
	Status        string     `json:"status"`
	PublishedAt   time.Time  `json:"published_at"`
	UnpublishedAt *time.Time `json:"unpublished_at"`
	RowVersion    uint32     `json:"row_version"`
}

type PublicScriptDetail struct {
	Script  Script  `json:"script"`
	Version Version `json:"version"`
	Content Content `json:"content"`
}
