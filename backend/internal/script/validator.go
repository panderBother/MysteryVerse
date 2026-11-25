package script

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Validator struct {
	now func() time.Time
}

func NewValidator() *Validator {
	return &Validator{now: time.Now}
}

func (v *Validator) Validate(
	ctx context.Context,
	repo storage.Repository,
	content Content,
) (ValidationReport, error) {
	report := ValidationReport{
		ScriptID:          content.Script.ID,
		ScriptVersionID:   content.Version.ID,
		VersionRowVersion: content.Version.RowVersion,
		Issues:            make([]ValidationIssue, 0),
		ValidatedAt:       v.now().UTC(),
	}
	v.validateMetadata(content, &report)
	v.validateChapters(content, &report)
	v.validateStages(content, &report)
	v.validateCharacters(content, &report)
	v.validateClues(content, &report)
	v.validateEndings(content, &report)
	v.validateAssets(content, &report)
	v.validateSearchPoints(content, &report)
	if err := v.validateEngine(ctx, repo, content, &report); err != nil {
		return ValidationReport{}, err
	}
	finalizeReport(&report)
	return report, nil
}

func (v *Validator) validateMetadata(content Content, report *ValidationReport) {
	script := content.Script
	if strings.TrimSpace(script.Title) == "" {
		addError(report, "SCRIPT_TITLE_REQUIRED", "script.title", "Script title is required.", nil)
	}
	if script.GameType != "script_murder" {
		addError(
			report,
			"UNSUPPORTED_GAME_TYPE",
			"script.game_type",
			"Only script_murder content can be published in this task.",
			nil,
		)
	}
	if utf8.RuneCountInString(script.Language) < 2 {
		addError(
			report,
			"SCRIPT_LANGUAGE_INVALID",
			"script.language",
			"Language code is invalid.",
			nil,
		)
	}
	if script.MinPlayers < 1 || script.MaxPlayers < script.MinPlayers {
		addError(
			report,
			"SCRIPT_PLAYER_RANGE_INVALID",
			"script.players",
			"Player range is invalid.",
			nil,
		)
	}
	if script.DurationMinutes < 1 {
		addError(
			report,
			"SCRIPT_DURATION_INVALID",
			"script.duration_minutes",
			"Duration must be positive.",
			nil,
		)
	}
	if !oneOf(script.ContentRating, "all", "12+", "16+", "18+") {
		addError(
			report,
			"SCRIPT_CONTENT_RATING_INVALID",
			"script.content_rating",
			"Content rating is invalid.",
			nil,
		)
	}
	if content.Version.SchemaVersion == "" {
		addError(
			report,
			"SCRIPT_SCHEMA_VERSION_REQUIRED",
			"version.schema_version",
			"Script schema version is required.",
			nil,
		)
	}
}

func (v *Validator) validateChapters(content Content, report *ValidationReport) {
	seenKeys := make(map[string]struct{}, len(content.Chapters))
	seenSequences := make(map[int]struct{}, len(content.Chapters))
	referenced := make(map[string]struct{}, len(content.Stages))
	for _, stage := range content.Stages {
		if stage.ChapterID != nil {
			referenced[*stage.ChapterID] = struct{}{}
		}
	}
	for index, chapter := range content.Chapters {
		path := fmt.Sprintf("chapters[%d]", index)
		if strings.TrimSpace(chapter.ChapterKey) == "" {
			addError(
				report,
				"CHAPTER_KEY_REQUIRED",
				path+".chapter_key",
				"Chapter key is required.",
				stringPointer(chapter.ID),
			)
		}
		if _, exists := seenKeys[chapter.ChapterKey]; exists {
			addError(
				report,
				"CHAPTER_KEY_DUPLICATE",
				path+".chapter_key",
				"Chapter key is duplicated.",
				stringPointer(chapter.ID),
			)
		}
		seenKeys[chapter.ChapterKey] = struct{}{}
		if _, exists := seenSequences[chapter.SequenceNo]; exists {
			addError(
				report,
				"CHAPTER_SEQUENCE_DUPLICATE",
				path+".sequence_no",
				"Chapter sequence is duplicated.",
				stringPointer(chapter.ID),
			)
		}
		seenSequences[chapter.SequenceNo] = struct{}{}
		if strings.TrimSpace(chapter.Title) == "" {
			addError(
				report,
				"CHAPTER_TITLE_REQUIRED",
				path+".title",
				"Chapter title is required.",
				stringPointer(chapter.ID),
			)
		}
		if _, exists := referenced[chapter.ID]; !exists {
			addWarning(
				report,
				"CHAPTER_UNREACHABLE",
				path,
				"Chapter is not referenced by any stage.",
				stringPointer(chapter.ID),
			)
		}
	}
}

func (v *Validator) validateStages(content Content, report *ValidationReport) {
	chapterIDs := make(map[string]struct{}, len(content.Chapters))
	for _, chapter := range content.Chapters {
		chapterIDs[chapter.ID] = struct{}{}
	}
	seenKeys := make(map[string]struct{}, len(content.Stages))
	seenSequences := make(map[int]struct{}, len(content.Stages))
	hasVote := false
	for index, stage := range content.Stages {
		path := fmt.Sprintf("stages[%d]", index)
		if strings.TrimSpace(stage.Key) == "" {
			addError(
				report,
				"STAGE_KEY_REQUIRED",
				path+".stage_key",
				"Stage key is required.",
				stringPointer(stage.ID),
			)
		}
		if _, exists := seenKeys[stage.Key]; exists {
			addError(
				report,
				"STAGE_KEY_DUPLICATE",
				path+".stage_key",
				"Stage key is duplicated.",
				stringPointer(stage.ID),
			)
		}
		seenKeys[stage.Key] = struct{}{}
		if _, exists := seenSequences[stage.SequenceNo]; exists {
			addError(
				report,
				"STAGE_SEQUENCE_DUPLICATE",
				path+".sequence_no",
				"Stage sequence is duplicated.",
				stringPointer(stage.ID),
			)
		}
		seenSequences[stage.SequenceNo] = struct{}{}
		if err := domain.PhaseKind(stage.Type).Validate(); err != nil {
			addError(
				report,
				"STAGE_PHASE_TYPE_UNSUPPORTED",
				path+".phase_type",
				err.Error(),
				stringPointer(stage.ID),
			)
		}
		if stage.Type == string(domain.PhaseVote) {
			hasVote = true
		}
		if len(stage.AllowedActions) == 0 {
			addError(
				report,
				"STAGE_ACTIONS_REQUIRED",
				path+".allowed_actions",
				"At least one allowed action is required.",
				stringPointer(stage.ID),
			)
		}
		for actionIndex, action := range stage.AllowedActions {
			if !oneOf(
				action,
				string(domain.ActionReadingComplete),
				string(domain.ActionSearch),
				string(domain.ActionVote),
				string(domain.ActionAdvancePhase),
			) {
				addError(
					report,
					"STAGE_ACTION_UNSUPPORTED",
					fmt.Sprintf("%s.allowed_actions[%d]", path, actionIndex),
					"Action is not supported by the current engine.",
					stringPointer(stage.ID),
				)
			}
		}
		if stage.ChapterID != nil {
			if _, exists := chapterIDs[*stage.ChapterID]; !exists {
				addError(
					report,
					"STAGE_CHAPTER_REFERENCE_INVALID",
					path+".chapter_id",
					"Stage references an unknown chapter.",
					stringPointer(stage.ID),
				)
			}
		}
		if nonEmptyRawObject(stage.EntryConditions) || nonEmptyRawObject(stage.ExitConditions) {
			addUnsupported(
				report,
				"UNSUPPORTED_PHASE_CONDITION_DSL",
				path+".entry_conditions",
				"Phase entry/exit condition DSL is not approved and cannot be published.",
				stringPointer(stage.ID),
			)
		}
		if stage.TimeoutSeconds != nil && *stage.TimeoutSeconds <= 0 {
			addError(
				report,
				"STAGE_TIMEOUT_INVALID",
				path+".timeout_seconds",
				"Stage timeout must be positive.",
				stringPointer(stage.ID),
			)
		}
	}
	if len(content.Stages) == 0 {
		addError(report, "STAGES_REQUIRED", "stages", "At least one stage is required.", nil)
	}
	if !hasVote {
		addError(report, "VOTE_STAGE_REQUIRED", "stages", "A vote stage is required.", nil)
	}
}

func (v *Validator) validateCharacters(content Content, report *ValidationReport) {
	playerRoles := 0
	seenKeys := make(map[string]struct{}, len(content.Characters))
	for index, character := range content.Characters {
		path := fmt.Sprintf("characters[%d]", index)
		if character.RoleClass == "player_role" {
			playerRoles++
		} else {
			addError(
				report,
				"ENGINE_UNSUPPORTED_CHARACTER_ROLE",
				path+".role_class",
				"The current engine can only load player_role characters.",
				stringPointer(character.ID),
			)
		}
		if _, exists := seenKeys[character.CharacterKey]; exists {
			addError(
				report,
				"CHARACTER_KEY_DUPLICATE",
				path+".character_key",
				"Character key is duplicated.",
				stringPointer(character.ID),
			)
		}
		seenKeys[character.CharacterKey] = struct{}{}
		if strings.TrimSpace(character.DisplayName) == "" {
			addError(
				report,
				"CHARACTER_NAME_REQUIRED",
				path+".display_name",
				"Character display name is required.",
				stringPointer(character.ID),
			)
		}
		if strings.TrimSpace(character.PublicProfile) == "" {
			addError(
				report,
				"CHARACTER_PUBLIC_PROFILE_REQUIRED",
				path+".public_profile",
				"Character public profile is required.",
				stringPointer(character.ID),
			)
		}
		if len(character.PublicGoals) == 0 {
			addWarning(
				report,
				"CHARACTER_GOALS_MISSING",
				path+".public_goals",
				"Character has no public goal configured.",
				stringPointer(character.ID),
			)
		}
	}
	if len(content.Characters) < content.Script.MinPlayers ||
		len(content.Characters) > content.Script.MaxPlayers {
		addError(
			report,
			"CHARACTER_COUNT_OUT_OF_RANGE",
			"characters",
			"Character count must fall within the script player range.",
			nil,
		)
	}
	if playerRoles != len(content.Characters) {
		addError(
			report,
			"CHARACTER_ROLE_CLASS_MISMATCH",
			"characters",
			"All configured characters must be player roles for the current engine.",
			nil,
		)
	}
	addWarning(
		report,
		"SECRET_PAYLOAD_AAD_PENDING",
		"characters.private_profile_ref",
		"Character private payload AAD remains unapproved; private content is not writable by this task.",
		nil,
	)
}

func (v *Validator) validateClues(content Content, report *ValidationReport) {
	linkedClues := make(map[string]struct{}, len(content.PointClues))
	for _, link := range content.PointClues {
		linkedClues[link.ClueID] = struct{}{}
	}
	seenKeys := make(map[string]struct{}, len(content.Clues))
	for index, clue := range content.Clues {
		path := fmt.Sprintf("clues[%d]", index)
		if _, exists := seenKeys[clue.ClueKey]; exists {
			addError(
				report,
				"CLUE_KEY_DUPLICATE",
				path+".clue_key",
				"Clue key is duplicated.",
				stringPointer(clue.ID),
			)
		}
		seenKeys[clue.ClueKey] = struct{}{}
		if !oneOf(clue.ClueType, "public", "private", "physical", "testimony") {
			addError(
				report,
				"CLUE_TYPE_INVALID",
				path+".clue_type",
				"Clue type is invalid.",
				stringPointer(clue.ID),
			)
		}
		if clue.PublicText == nil && clue.PrivateTextRef == nil {
			addError(
				report,
				"CLUE_CONTENT_REQUIRED",
				path,
				"Clue must have public text or a private content reference.",
				stringPointer(clue.ID),
			)
		}
		if clue.IsRepeatable && clue.MaxDiscoveries != nil {
			addError(
				report,
				"CLUE_REPEATABLE_LIMIT_CONFLICT",
				path+".max_discoveries",
				"A repeatable clue cannot have a finite max_discoveries.",
				stringPointer(clue.ID),
			)
		}
		if !clue.IsRepeatable && (clue.MaxDiscoveries == nil || *clue.MaxDiscoveries != 1) {
			addError(
				report,
				"CLUE_SINGLE_DISCOVERY_RULE",
				path+".max_discoveries",
				"A non-repeatable clue must have max_discoveries equal to 1.",
				stringPointer(clue.ID),
			)
		}
		if clue.Visibility != "public" {
			if _, exists := linkedClues[clue.ID]; !exists {
				addError(
					report,
					"CLUE_UNREACHABLE",
					path,
					"Non-public clue is not linked to any search point.",
					stringPointer(clue.ID),
				)
			}
		}
	}
}

func (v *Validator) validateEndings(content Content, report *ValidationReport) {
	seenKeys := make(map[string]struct{}, len(content.Endings))
	for index, ending := range content.Endings {
		path := fmt.Sprintf("endings[%d]", index)
		if _, exists := seenKeys[ending.EndingKey]; exists {
			addError(
				report,
				"ENDING_KEY_DUPLICATE",
				path+".ending_key",
				"Ending key is duplicated.",
				stringPointer(ending.ID),
			)
		}
		seenKeys[ending.EndingKey] = struct{}{}
		if strings.TrimSpace(ending.Title) == "" {
			addError(
				report,
				"ENDING_TITLE_REQUIRED",
				path+".title",
				"Ending title is required.",
				stringPointer(ending.ID),
			)
		}
		if nonEmptyRawObject(&ending.Conditions) {
			addUnsupported(
				report,
				"UNSUPPORTED_ENDING_CONDITION_DSL",
				path+".conditions",
				"Combined ending conditions are not approved and cannot be published.",
				stringPointer(ending.ID),
			)
		}
	}
	if len(content.Endings) == 0 {
		addWarning(
			report,
			"ENDINGS_NOT_CONFIGURED",
			"endings",
			"No narrative ending is configured; settlement currently relies on the rule template.",
			nil,
		)
	}
}

func (v *Validator) validateAssets(content Content, report *ValidationReport) {
	seenKeys := make(map[string]struct{}, len(content.Assets))
	assetURIs := make(map[string]struct{}, len(content.Assets))
	for index, asset := range content.Assets {
		path := fmt.Sprintf("assets[%d]", index)
		if _, exists := seenKeys[asset.AssetKey]; exists {
			addError(
				report,
				"ASSET_KEY_DUPLICATE",
				path+".asset_key",
				"Asset key is duplicated.",
				stringPointer(asset.ID),
			)
		}
		seenKeys[asset.AssetKey] = struct{}{}
		assetURIs[asset.URI] = struct{}{}
		if err := validateAssetURI(asset.URI); err != nil {
			addError(
				report,
				"ASSET_URI_INVALID",
				path+".uri",
				err.Error(),
				stringPointer(asset.ID),
			)
		}
		if asset.SHA256 != nil && !sha256Pattern.MatchString(*asset.SHA256) {
			addError(
				report,
				"ASSET_HASH_INVALID",
				path+".sha256",
				"Asset sha256 must be 64 lowercase hexadecimal characters.",
				stringPointer(asset.ID),
			)
		}
		if asset.ByteSize != nil && *asset.ByteSize < 1 {
			addError(
				report,
				"ASSET_SIZE_INVALID",
				path+".byte_size",
				"Asset byte size must be positive.",
				stringPointer(asset.ID),
			)
		}
	}
	if content.Script.CoverAssetURI != nil {
		if _, exists := assetURIs[*content.Script.CoverAssetURI]; !exists {
			addError(
				report,
				"COVER_ASSET_REFERENCE_INVALID",
				"script.cover_asset_uri",
				"Cover asset is not registered in the script version assets.",
				nil,
			)
		}
	}
}

func (v *Validator) validateSearchPoints(content Content, report *ValidationReport) {
	stages := make(map[string]Stage, len(content.Stages))
	points := make(map[string]SearchPoint, len(content.SearchPoints))
	clues := make(map[string]Clue, len(content.Clues))
	hasSearchStage := false
	for _, stage := range content.Stages {
		stages[stage.ID] = stage
		if stage.Type == string(domain.PhaseSearch) {
			hasSearchStage = true
		}
	}
	for _, clue := range content.Clues {
		clues[clue.ID] = clue
	}
	seenKeys := make(map[string]struct{}, len(content.SearchPoints))
	for index, point := range content.SearchPoints {
		points[point.ID] = point
		path := fmt.Sprintf("search_points[%d]", index)
		if _, exists := seenKeys[point.PointKey]; exists {
			addError(
				report,
				"SEARCH_POINT_KEY_DUPLICATE",
				path+".point_key",
				"Search point key is duplicated.",
				stringPointer(point.ID),
			)
		}
		seenKeys[point.PointKey] = struct{}{}
		stage, exists := stages[point.StageID]
		if !exists {
			addError(
				report,
				"SEARCH_POINT_STAGE_INVALID",
				path+".stage_id",
				"Search point references an unknown stage.",
				stringPointer(point.ID),
			)
		} else if stage.Type != string(domain.PhaseSearch) {
			addError(
				report,
				"SEARCH_POINT_STAGE_NOT_SEARCH",
				path+".stage_id",
				"Search point must reference a search stage.",
				stringPointer(point.ID),
			)
		}
		if point.MaxSearches < 1 {
			addError(
				report,
				"SEARCH_POINT_LIMIT_INVALID",
				path+".max_searches",
				"Search point max_searches must be positive.",
				stringPointer(point.ID),
			)
		}
		if !oneOf(point.ConflictPolicy, "first_wins", "repeatable", "queue") {
			addError(
				report,
				"SEARCH_POINT_POLICY_INVALID",
				path+".conflict_policy",
				"Search point conflict policy is invalid.",
				stringPointer(point.ID),
			)
		}
	}
	if hasSearchStage && len(content.SearchPoints) == 0 {
		addError(
			report,
			"SEARCH_POINTS_REQUIRED",
			"search_points",
			"A search stage requires at least one search point.",
			nil,
		)
	}
	seenLinks := make(map[string]struct{}, len(content.PointClues))
	for index, link := range content.PointClues {
		path := fmt.Sprintf("point_clues[%d]", index)
		if _, exists := points[link.SearchPointID]; !exists {
			addError(
				report,
				"SEARCH_POINT_CLUE_POINT_INVALID",
				path+".search_point_id",
				"Search point clue references an unknown search point.",
				nil,
			)
		}
		if _, exists := clues[link.ClueID]; !exists {
			addError(
				report,
				"SEARCH_POINT_CLUE_REFERENCE_INVALID",
				path+".clue_id",
				"Search point clue references an unknown clue.",
				nil,
			)
		}
		key := link.SearchPointID + ":" + link.ClueID
		if _, exists := seenLinks[key]; exists {
			addError(
				report,
				"SEARCH_POINT_CLUE_DUPLICATE",
				path,
				"Search point clue relation is duplicated.",
				nil,
			)
		}
		seenLinks[key] = struct{}{}
		if link.Weight < 1 {
			addError(
				report,
				"SEARCH_POINT_CLUE_WEIGHT_INVALID",
				path+".weight",
				"Search point clue weight must be positive.",
				nil,
			)
		}
	}
}

func (v *Validator) validateEngine(
	ctx context.Context,
	repo storage.Repository,
	content Content,
	report *ValidationReport,
) error {
	if content.Version.RuleTemplateID == nil {
		addError(
			report,
			"ENGINE_RULE_TEMPLATE_REQUIRED",
			"version.rule_template_id",
			"A controlled rule template is required before publishing.",
			nil,
		)
		return nil
	}
	templateRow, err := repo.GetActiveRuleTemplateForScript(
		ctx,
		sqlc.GetActiveRuleTemplateForScriptParams{
			ID:       *content.Version.RuleTemplateID,
			GameType: content.Script.GameType,
		},
	)
	if err != nil {
		addError(
			report,
			"ENGINE_RULE_TEMPLATE_UNAVAILABLE",
			"version.rule_template_id",
			"The active rule template is unavailable.",
			nil,
		)
		return nil
	}
	template := ruleTemplateFromRow(templateRow)
	if template.EngineVersion != domain.EngineVersion {
		addError(
			report,
			"ENGINE_VERSION_INCOMPATIBLE",
			"version.rule_template_id",
			"The rule template engine version is not supported.",
			nil,
		)
		return nil
	}
	policy, err := parseMurderPolicy(template.Config)
	if err != nil {
		addError(
			report,
			"ENGINE_POLICY_INVALID",
			"version.rule_template_id",
			err.Error(),
			nil,
		)
		return nil
	}
	config, err := validatorEngineConfig(content, policy)
	if err != nil {
		addError(
			report,
			"ENGINE_CONTENT_INCOMPATIBLE",
			"content",
			err.Error(),
			nil,
		)
		return nil
	}
	if _, err := domain.NewScriptMurderEngine(config); err != nil {
		addError(
			report,
			"ENGINE_CONTENT_INCOMPATIBLE",
			"content",
			err.Error(),
			nil,
		)
	}
	addWarning(
		report,
		"VOTE_POLICY_SCHEMA_PENDING",
		"version.rule_template_id",
		"Voting policy schema is not frozen; only the explicitly supported engine policy can be published.",
		nil,
	)
	return nil
}

func validatorEngineConfig(content Content, policy domain.MurderPolicy) (domain.ScriptConfig, error) {
	config := domain.ScriptConfig{
		ScriptVersionID: content.Version.ID,
		Title:           content.Script.Title,
		EngineKey:       domain.EngineKey,
		EngineVersion:   domain.EngineVersion,
		SchemaVersion:   content.Version.SchemaVersion,
		Policy:          policy,
		Phases:          make([]domain.Phase, 0, len(content.Stages)),
		Characters:      make([]domain.Character, 0, len(content.Characters)),
		Clues:           make([]domain.Clue, 0, len(content.Clues)),
		SearchPoints:    make([]domain.SearchPoint, 0, len(content.SearchPoints)),
		PointClues:      make([]domain.SearchPointClue, 0, len(content.PointClues)),
	}
	for _, stage := range content.Stages {
		actions := make([]domain.ActionType, 0, len(stage.AllowedActions))
		for _, action := range stage.AllowedActions {
			actions = append(actions, domain.ActionType(action))
		}
		config.Phases = append(config.Phases, domain.Phase{
			ID:             stage.ID,
			Key:            stage.Key,
			Kind:           domain.PhaseKind(stage.Type),
			Sequence:       stage.SequenceNo,
			Title:          stage.Title,
			AllowedActions: actions,
			TimeoutSeconds: stage.TimeoutSeconds,
			FallbackAction: stage.FallbackAction,
			IsTerminal:     stage.IsTerminal,
		})
	}
	for _, character := range content.Characters {
		config.Characters = append(config.Characters, domain.Character{
			ID:               character.ID,
			Key:              character.CharacterKey,
			DisplayName:      character.DisplayName,
			RoleClass:        character.RoleClass,
			PublicProfile:    character.PublicProfile,
			PublicGoals:      mustJSON(character.PublicGoals),
			Attributes:       rawJSON(character.Attributes),
			PrivatePayloadID: stringValue(character.PrivatePayloadID),
		})
	}
	for _, clue := range content.Clues {
		value := domain.Clue{
			ID:               clue.ID,
			Key:              clue.ClueKey,
			Title:            clue.Title,
			ClueType:         clue.ClueType,
			IsRepeatable:     clue.IsRepeatable,
			MaxDiscoveries:   clue.MaxDiscoveries,
			VisibilityScope:  clue.Visibility,
			Sensitivity:      int(sensitivityNumber(clue.Sensitivity)),
			PrivatePayloadID: stringValue(clue.PrivatePayloadID),
		}
		if clue.PublicText != nil {
			value.PublicText = *clue.PublicText
		}
		config.Clues = append(config.Clues, value)
	}
	stageKeys := make(map[string]string, len(content.Stages))
	for _, stage := range content.Stages {
		stageKeys[stage.ID] = stage.Key
	}
	for _, point := range content.SearchPoints {
		config.SearchPoints = append(config.SearchPoints, domain.SearchPoint{
			ID:                point.ID,
			Key:               point.PointKey,
			PhaseKey:          stageKeys[point.StageID],
			Title:             point.Title,
			PublicDescription: point.PublicDescription,
			MaxSearches:       point.MaxSearches,
			ConflictPolicy:    point.ConflictPolicy,
		})
	}
	for _, link := range content.PointClues {
		config.PointClues = append(config.PointClues, domain.SearchPointClue{
			SearchPointID: link.SearchPointID,
			ClueID:        link.ClueID,
			Weight:        link.Weight,
			IsDefault:     link.IsDefault,
		})
	}
	return config, nil
}

func parseMurderPolicy(raw json.RawMessage) (domain.MurderPolicy, error) {
	var envelope struct {
		SchemaVersion string              `json:"schema_version"`
		Murder        domain.MurderPolicy `json:"murder"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.MurderPolicy{}, fmt.Errorf("rule template config is invalid: %w", err)
	}
	policy := envelope.Murder
	if policy.SchemaVersion == "" {
		policy.SchemaVersion = envelope.SchemaVersion
	}
	if err := policy.Validate(); err != nil {
		return domain.MurderPolicy{}, fmt.Errorf("murder policy is invalid: %w", err)
	}
	return policy, nil
}

func finalizeReport(report *ValidationReport) {
	report.BlockingCount = 0
	report.WarningCount = 0
	report.InfoCount = 0
	for _, issue := range report.Issues {
		switch issue.Severity {
		case SeverityError:
			report.BlockingCount++
		case SeverityWarning:
			report.WarningCount++
		case SeverityInfo:
			report.InfoCount++
		}
	}
	switch {
	case report.BlockingCount > 0:
		report.Status = "blocked"
	case report.WarningCount > 0:
		report.Status = "warning"
	default:
		report.Status = "passed"
	}
}

func addError(
	report *ValidationReport,
	code string,
	path string,
	message string,
	objectID *string,
) {
	report.Issues = append(report.Issues, ValidationIssue{
		ErrorCode: code,
		Severity:  SeverityError,
		Path:      path,
		Message:   message,
		State:     ValidationStateChecked,
		ObjectID:  objectID,
	})
}

func addWarning(
	report *ValidationReport,
	code string,
	path string,
	message string,
	objectID *string,
) {
	report.Issues = append(report.Issues, ValidationIssue{
		ErrorCode: code,
		Severity:  SeverityWarning,
		Path:      path,
		Message:   message,
		State:     ValidationStateChecked,
		ObjectID:  objectID,
	})
}

func addUnsupported(
	report *ValidationReport,
	code string,
	path string,
	message string,
	objectID *string,
) {
	report.Issues = append(report.Issues, ValidationIssue{
		ErrorCode: code,
		Severity:  SeverityError,
		Path:      path,
		Message:   message,
		State:     ValidationStateUnsupported,
		ObjectID:  objectID,
	})
}

func nonEmptyRawObject(raw *json.RawMessage) bool {
	if raw == nil || len(*raw) == 0 {
		return false
	}
	var value any
	if err := json.Unmarshal(*raw, &value); err != nil {
		return true
	}
	switch typed := value.(type) {
	case nil:
		return false
	case map[string]any:
		return len(typed) > 0
	case []any:
		return len(typed) > 0
	default:
		return true
	}
}

func validateAssetURI(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("asset uri is invalid")
	}
	switch parsed.Scheme {
	case "s3", "https":
		if parsed.Host == "" && parsed.Opaque == "" {
			return fmt.Errorf("asset uri must include a controlled host or bucket")
		}
		return nil
	default:
		return fmt.Errorf("asset uri scheme is not allowed")
	}
}

func findSearchPoint(points []SearchPoint, id string) (SearchPoint, bool) {
	for _, point := range points {
		if point.ID == id {
			return point, true
		}
	}
	return SearchPoint{}, false
}

func stringPointer(value string) *string {
	copyValue := value
	return &copyValue
}
