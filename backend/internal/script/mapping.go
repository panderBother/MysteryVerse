package script

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"

	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

func scriptFromRow(row sqlc.Script) Script {
	return Script{
		ID:                 row.ID,
		OwnerUserID:        row.OwnerUserID,
		PublicSlug:         nullStringPointer(row.PublicSlug),
		Title:              row.Title,
		Summary:            nullStringPointer(row.Summary),
		GameType:           row.GameType,
		SourceKind:         row.SourceKind,
		RightsType:         row.RightsType,
		Status:             row.Status,
		CoverAssetURI:      nullStringPointer(row.CoverAssetUri),
		Language:           row.Language,
		ContentRating:      row.ContentRating,
		MinPlayers:         int(row.MinPlayers),
		MaxPlayers:         int(row.MaxPlayers),
		DurationMinutes:    int(row.DurationMinutes),
		Difficulty:         row.Difficulty,
		AIAssistedRatioBP:  int(row.AiAssistedRatioBp),
		ActiveDraftVersion: nullStringPointer(row.ActiveDraftVersionID),
		PublishedVersion:   nullStringPointer(row.PublishedVersionID),
		RowVersion:         row.RowVersion,
		DeletedAt:          nullTimePointer(row.DeletedAt),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func versionFromRow(row sqlc.ScriptVersion) Version {
	return Version{
		ID:                 row.ID,
		ScriptID:           row.ScriptID,
		VersionNo:          int(row.VersionNo),
		VersionKind:        row.VersionKind,
		Status:             row.Status,
		SchemaVersion:      row.SchemaVersion,
		WorkflowVersion:    row.WorkflowVersion,
		BaseVersionID:      nullStringPointer(row.BaseVersionID),
		RuleTemplateID:     nullStringPointer(row.RuleTemplateID),
		Blueprint:          append(json.RawMessage(nil), row.BlueprintJson...),
		PackageURI:         nullStringPointer(row.PackageUri),
		PackageHash:        nullStringBytes(row.PackageHash),
		Manifest:           rawMessageValue(row.ManifestJson),
		ValidationReport:   rawMessageValue(row.ValidationReportJson),
		ValidationStatus:   row.ValidationStatus,
		BlockingIssueCount: int(row.BlockingIssueCount),
		CreatedBy:          row.CreatedBy,
		FrozenAt:           nullTimePointer(row.FrozenAt),
		PublishedAt:        nullTimePointer(row.PublishedAt),
		RowVersion:         row.RowVersion,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func chapterFromRow(row sqlc.ScriptChapter) Chapter {
	return Chapter{
		ID:         row.ID,
		ChapterKey: row.ChapterKey,
		Title:      row.Title,
		SequenceNo: int(row.SequenceNo),
		Summary:    nullStringPointer(row.Summary),
		SourceRefs: append(json.RawMessage(nil), row.SourceRefsJson...),
		RowVersion: row.RowVersion,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func stageFromRow(row sqlc.ScriptPhase) Stage {
	var actions []string
	_ = json.Unmarshal(row.AllowedActionsJson, &actions)
	return Stage{
		ID:              row.ID,
		ChapterID:       nullStringPointer(row.ChapterID),
		Key:             row.PhaseKey,
		Type:            row.PhaseType,
		SequenceNo:      int(row.SequenceNo),
		Title:           row.Title,
		EntryConditions: rawMessagePointerValue(row.EntryConditionJson),
		ExitConditions:  rawMessagePointerValue(row.ExitConditionJson),
		TimeoutSeconds:  nullInt32Pointer(row.TimeoutSeconds),
		FallbackAction:  row.FallbackAction,
		AllowedActions:  actions,
		Visibility:      row.VisibilityScope,
		IsTerminal:      row.IsTerminal,
		RowVersion:      row.RowVersion,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func characterFromRow(row sqlc.ScriptCharacter) Character {
	var goals []string
	_ = json.Unmarshal(row.PublicGoalsJson, &goals)
	var locked []string
	if row.LockedFieldsJson != nil {
		_ = json.Unmarshal(*row.LockedFieldsJson, &locked)
	}
	return Character{
		ID:               row.ID,
		CharacterKey:     row.CharacterKey,
		DisplayName:      row.DisplayName,
		RoleClass:        row.RoleClass,
		PublicProfile:    row.PublicProfile,
		PublicGoals:      goals,
		Attributes:       append(json.RawMessage(nil), row.AttributesJson...),
		Visibility:       row.VisibilityScope,
		Sensitivity:      sensitivityName(row.Sensitivity),
		SourceRefs:       rawMessageValue(row.SourceRefsJson),
		GeneratedOrigin:  row.GeneratedOrigin,
		LockedFields:     locked,
		SortOrder:        int(row.SortOrder),
		RowVersion:       row.RowVersion,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		PrivatePayloadID: nullStringPointer(row.PrivatePayloadID),
	}
}

func clueFromRow(row sqlc.ScriptClue) Clue {
	var facts []string
	if row.RelatedFactIdsJson != nil {
		_ = json.Unmarshal(*row.RelatedFactIdsJson, &facts)
	}
	return Clue{
		ID:               row.ID,
		ClueKey:          row.ClueKey,
		Title:            row.Title,
		PublicText:       nullStringPointer(row.PublicText),
		ClueType:         row.ClueType,
		IsRepeatable:     row.IsRepeatable,
		MaxDiscoveries:   nullInt32Pointer(row.MaxDiscoveries),
		Visibility:       row.VisibilityScope,
		Sensitivity:      sensitivityName(row.Sensitivity),
		RevealGateID:     nullStringPointer(row.RevealGateID),
		RelatedFactIDs:   facts,
		SourceRefs:       rawMessageValue(row.SourceRefsJson),
		SortOrder:        int(row.SortOrder),
		RowVersion:       row.RowVersion,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		PrivatePayloadID: nullStringPointer(row.PrivatePayloadID),
	}
}

func endingFromRow(row sqlc.ScriptEnding) Ending {
	return Ending{
		ID:          row.ID,
		EndingKey:   row.EndingKey,
		Title:       row.Title,
		Description: row.Description,
		Conditions:  append(json.RawMessage(nil), row.ConditionsJson...),
		IsHidden:    row.IsHidden,
		Visibility:  row.VisibilityScope,
		SortOrder:   int(row.SortOrder),
		RowVersion:  row.RowVersion,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func assetFromRow(row sqlc.ScriptAsset) Asset {
	var sha *string
	if row.Sha256.Valid {
		value := hex.EncodeToString([]byte(row.Sha256.String))
		sha = &value
	}
	return Asset{
		ID:          row.ID,
		AssetKey:    row.AssetKey,
		AssetType:   row.AssetType,
		URI:         row.Uri,
		MIMEType:    nullStringPointer(row.MimeType),
		SHA256:      sha,
		ByteSize:    nullInt64Pointer(row.ByteSize),
		Visibility:  row.VisibilityScope,
		Sensitivity: sensitivityName(row.Sensitivity),
		Metadata:    rawMessageValue(row.MetadataJson),
		SortOrder:   int(row.SortOrder),
		RowVersion:  row.RowVersion,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func searchPointFromRow(row sqlc.ScriptSearchPoint) SearchPoint {
	return SearchPoint{
		ID:                row.ID,
		PointKey:          row.PointKey,
		StageID:           row.PhaseID,
		SceneID:           nullStringPointer(row.SceneID),
		Title:             row.Title,
		PublicDescription: row.PublicDescription,
		MaxSearches:       int(row.MaxSearches),
		ConflictPolicy:    row.ConflictPolicy,
		Visibility:        row.VisibilityScope,
		RevealGateID:      nullStringPointer(row.RevealGateID),
		Metadata:          rawMessageValue(row.MetadataJson),
		RowVersion:        row.RowVersion,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func searchPointClueFromRow(row sqlc.ScriptSearchPointClue) SearchPointClue {
	return SearchPointClue{
		SearchPointID: row.SearchPointID,
		ClueID:        row.ClueID,
		Weight:        int(row.Weight),
		IsDefault:     row.IsDefault,
		Metadata:      rawMessageValue(row.MetadataJson),
		CreatedAt:     row.CreatedAt,
	}
}

func ruleTemplateFromRow(row sqlc.RuleTemplate) RuleTemplate {
	return RuleTemplate{
		ID:            row.ID,
		TemplateKey:   row.TemplateKey,
		VersionNo:     int(row.VersionNo),
		GameType:      row.GameType,
		Name:          row.Name,
		Status:        row.Status,
		SchemaVersion: row.SchemaVersion,
		EngineVersion: row.EngineVersion,
		Config:        append(json.RawMessage(nil), row.ConfigJson...),
		ContentHash:   append([]byte(nil), row.ContentHash...),
	}
}

func sensitivityName(value uint8) string {
	switch value {
	case 0:
		return "D0"
	case 1:
		return "D1"
	case 2:
		return "D2"
	case 3:
		return "D3"
	default:
		return "D4"
	}
}

func sensitivityNumber(value string) uint8 {
	switch value {
	case "D0":
		return 0
	case "D1":
		return 1
	case "D3":
		return 3
	case "D4":
		return 4
	default:
		return 2
	}
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copyValue := value.String
	return &copyValue
}

func nullInt32Pointer(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	copyValue := int(value.Int32)
	return &copyValue
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	copyValue := value.Int64
	return &copyValue
}

func rawMessageValue(value any) json.RawMessage {
	switch typed := value.(type) {
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...)
	case *json.RawMessage:
		if typed == nil {
			return json.RawMessage("null")
		}
		return append(json.RawMessage(nil), (*typed)...)
	case []byte:
		if typed == nil {
			return json.RawMessage("null")
		}
		return append(json.RawMessage(nil), typed...)
	default:
		return json.RawMessage("null")
	}
}

func rawMessagePointerValue(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	copyValue := append(json.RawMessage(nil), (*value)...)
	return &copyValue
}

func nullStringBytes(value sql.NullString) []byte {
	if !value.Valid {
		return nil
	}
	return []byte(value.String)
}

func rawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("{}")
	}
	return append(json.RawMessage(nil), value...)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
