package script

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

var supportedObjectTypes = map[string]struct{}{
	"chapter":           {},
	"stage":             {},
	"character":         {},
	"clue":              {},
	"ending":            {},
	"asset":             {},
	"search_point":      {},
	"search_point_clue": {},
}

func (s *Service) CreateObject(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	objectType string,
	input ObjectMutationInput,
	idempotencyKey string,
) (ObjectMutationResult, error) {
	return s.mutateObject(
		ctx,
		actorUserID,
		scriptID,
		versionID,
		objectType,
		"",
		input,
		false,
		idempotencyKey,
	)
}

func (s *Service) UpdateObject(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	objectType string,
	objectID string,
	input ObjectMutationInput,
	idempotencyKey string,
) (ObjectMutationResult, error) {
	return s.mutateObject(
		ctx,
		actorUserID,
		scriptID,
		versionID,
		objectType,
		objectID,
		input,
		true,
		idempotencyKey,
	)
}

func (s *Service) DeleteObject(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	objectType string,
	objectID string,
	expectedRowVersion uint32,
	idempotencyKey string,
) (ObjectMutationResult, error) {
	if idempotencyKey == "" {
		return ObjectMutationResult{}, validationError("Idempotency-Key is required")
	}
	if _, ok := supportedObjectTypes[objectType]; !ok {
		return ObjectMutationResult{}, unsupportedObjectTypeError(objectType)
	}
	if objectType == "search_point_clue" {
		return ObjectMutationResult{}, apperror.New(
			"FORBIDDEN_ACTION",
			"Search point clue relations must be created or updated through their composite object.",
			false,
			nil,
		)
	}
	requestHash := hashJSON(map[string]any{
		"object_type": objectType,
		"object_id":   objectID,
		"row_version": expectedRowVersion,
		"delete":      true,
	})
	var result ObjectMutationResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.object.delete",
			"user:"+actorUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		versionRow, err := s.lockDraft(ctx, tx, actorUserID, scriptID, versionID, expectedRowVersion)
		if err != nil {
			return err
		}
		if replay {
			result = ObjectMutationResult{
				ObjectType:        objectType,
				ObjectID:          objectID,
				Object:            map[string]any{"deleted": true},
				VersionRowVersion: versionRow.RowVersion,
			}
			return nil
		}
		if err := s.deleteObjectRow(ctx, tx, versionID, objectType, objectID); err != nil {
			return err
		}
		if err := tx.BumpScriptDraftRowVersion(ctx, sqlc.BumpScriptDraftRowVersionParams{
			ID:         versionID,
			ScriptID:   scriptID,
			RowVersion: versionRow.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script_version", Valid: true},
			ResourceID:     sql.NullString{String: versionID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		result = ObjectMutationResult{
			ObjectType:        objectType,
			ObjectID:          objectID,
			Object:            map[string]any{"deleted": true},
			VersionRowVersion: versionRow.RowVersion + 1,
		}
		return nil
	})
	return result, err
}

func (s *Service) mutateObject(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	objectType string,
	objectID string,
	input ObjectMutationInput,
	update bool,
	idempotencyKey string,
) (ObjectMutationResult, error) {
	if idempotencyKey == "" {
		return ObjectMutationResult{}, validationError("Idempotency-Key is required")
	}
	if _, ok := supportedObjectTypes[objectType]; !ok {
		return ObjectMutationResult{}, unsupportedObjectTypeError(objectType)
	}
	if input.ExpectedRowVersion == 0 {
		return ObjectMutationResult{}, validationError("expected_row_version is required")
	}
	if len(input.Object) == 0 {
		return ObjectMutationResult{}, validationError("object is required")
	}
	requestHash := hashJSON(map[string]any{
		"object_type": objectType,
		"object_id":   objectID,
		"row_version": input.ExpectedRowVersion,
		"object":      json.RawMessage(input.Object),
		"update":      update,
	})
	var result ObjectMutationResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.object.mutate",
			"user:"+actorUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		versionRow, err := s.lockDraft(
			ctx,
			tx,
			actorUserID,
			scriptID,
			versionID,
			input.ExpectedRowVersion,
		)
		if err != nil {
			return err
		}
		if replay {
			result = ObjectMutationResult{
				ObjectType:        objectType,
				ObjectID:          objectID,
				Object:            map[string]any{"replayed": true},
				VersionRowVersion: versionRow.RowVersion,
			}
			return nil
		}
		mutatedID, object, err := s.writeObject(
			ctx,
			tx,
			versionID,
			objectType,
			objectID,
			input.Object,
			update,
		)
		if err != nil {
			return err
		}
		if err := tx.BumpScriptDraftRowVersion(ctx, sqlc.BumpScriptDraftRowVersionParams{
			ID:         versionID,
			ScriptID:   scriptID,
			RowVersion: versionRow.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{
				Int16: objectMutationResponseStatus(!update),
				Valid: true,
			},
			ResourceType: sql.NullString{String: "script_version", Valid: true},
			ResourceID:   sql.NullString{String: versionID, Valid: true},
			ID:           record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		result = ObjectMutationResult{
			ObjectType:        objectType,
			ObjectID:          mutatedID,
			Object:            object,
			VersionRowVersion: versionRow.RowVersion + 1,
		}
		return nil
	})
	return result, err
}

func (s *Service) lockDraft(
	ctx context.Context,
	tx storage.Repository,
	actorUserID string,
	scriptID string,
	versionID string,
	expectedRowVersion uint32,
) (sqlc.ScriptVersion, error) {
	scriptRow, err := tx.GetScript(ctx, scriptID)
	if err != nil {
		return sqlc.ScriptVersion{}, scriptNotFoundError()
	}
	if scriptRow.OwnerUserID != actorUserID {
		return sqlc.ScriptVersion{}, forbiddenEditError()
	}
	versionRow, err := tx.GetScriptVersionForUpdate(ctx, versionID)
	if err != nil || versionRow.ScriptID != scriptID {
		return sqlc.ScriptVersion{}, versionNotFoundError()
	}
	if versionRow.Status != "draft" {
		return sqlc.ScriptVersion{}, versionFrozenError("Only draft versions can be edited.")
	}
	if versionRow.RowVersion != expectedRowVersion {
		return sqlc.ScriptVersion{}, versionConflictError()
	}
	return versionRow, nil
}

func (s *Service) writeObject(
	ctx context.Context,
	repo storage.Repository,
	versionID string,
	objectType string,
	objectID string,
	raw json.RawMessage,
	update bool,
) (string, any, error) {
	switch objectType {
	case "chapter":
		var input ChapterInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		chapterKey := strings.TrimSpace(input.ChapterKey)
		if chapterKey == "" {
			chapterKey = newID
		}
		sourceRefs := rawJSON(input.SourceRefs)
		if len(input.SourceRefs) == 0 {
			sourceRefs = json.RawMessage("[]")
		}
		params := sqlc.CreateScriptChapterParams{
			ID:              newID,
			ScriptVersionID: versionID,
			ChapterKey:      chapterKey,
			Title:           strings.TrimSpace(input.Title),
			Summary:         nullableStringParam(input.Summary),
			SequenceNo:      uint32(input.SequenceNo),
			SourceRefsJson:  sourceRefs,
		}
		if params.Title == "" {
			return "", nil, validationError("chapter title is required")
		}
		if update {
			if err := repo.UpdateScriptChapter(ctx, sqlc.UpdateScriptChapterParams{
				ChapterKey:      params.ChapterKey,
				Title:           params.Title,
				Summary:         params.Summary,
				SequenceNo:      params.SequenceNo,
				SourceRefsJson:  params.SourceRefsJson,
				ID:              newID,
				ScriptVersionID: versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptChapter(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptChapter(ctx, sqlc.GetScriptChapterParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, chapterFromRow(row), nil
	case "stage":
		var input StageInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptStageParams{
			ID:                 newID,
			ScriptVersionID:    versionID,
			ChapterID:          nullableStringParam(input.ChapterID),
			PhaseKey:           strings.TrimSpace(input.StageKey),
			PhaseType:          strings.TrimSpace(input.PhaseType),
			SequenceNo:         uint32(input.SequenceNo),
			Title:              strings.TrimSpace(input.Title),
			EntryConditionJson: rawMessagePointerInput(input.EntryConditions),
			ExitConditionJson:  rawMessagePointerInput(input.ExitConditions),
			TimeoutSeconds:     nullableInt32Param(input.TimeoutSeconds),
			FallbackAction:     strings.TrimSpace(input.FallbackAction),
			AllowedActionsJson: mustJSON(input.AllowedActions),
			VisibilityScope:    defaultVisibility(input.Visibility, "room"),
			IsTerminal:         input.IsTerminal,
		}
		if params.PhaseKey == "" || params.PhaseType == "" || params.Title == "" {
			return "", nil, validationError("stage_key, phase_type, and title are required")
		}
		if !oneOf(params.PhaseType, "read", "opening", "search", "private_chat",
			"discussion", "final_statement", "vote", "settlement", "replay") {
			return "", nil, validationError("phase_type is invalid")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if params.ChapterID.Valid && id.Validate(params.ChapterID.String) != nil {
			return "", nil, validationError("chapter_id must be a ULID")
		}
		for _, action := range input.AllowedActions {
			if !oneOf(action, "reading_complete", "search", "vote", "advance_phase") {
				return "", nil, validationError("allowed_actions contains an unsupported action")
			}
		}
		if update {
			if err := repo.UpdateScriptStage(ctx, sqlc.UpdateScriptStageParams{
				ChapterID:          params.ChapterID,
				PhaseKey:           params.PhaseKey,
				PhaseType:          params.PhaseType,
				SequenceNo:         params.SequenceNo,
				Title:              params.Title,
				EntryConditionJson: params.EntryConditionJson,
				ExitConditionJson:  params.ExitConditionJson,
				TimeoutSeconds:     params.TimeoutSeconds,
				FallbackAction:     params.FallbackAction,
				AllowedActionsJson: params.AllowedActionsJson,
				VisibilityScope:    params.VisibilityScope,
				IsTerminal:         params.IsTerminal,
				ID:                 newID,
				ScriptVersionID:    versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptStage(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptStage(ctx, sqlc.GetScriptStageParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, stageFromRow(row), nil
	case "character":
		var input CharacterInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if input.PrivateProfileRef != nil {
			return "", nil, apperror.New(
				"VALIDATION_ERROR",
				"private_profile_ref cannot be written until SecretPayload AAD is approved.",
				false,
				nil,
			)
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptCharacterParams{
			ID:               newID,
			ScriptVersionID:  versionID,
			CharacterKey:     strings.TrimSpace(input.CharacterKey),
			DisplayName:      strings.TrimSpace(input.DisplayName),
			RoleClass:        defaultString(input.RoleClass, "player_role"),
			PublicProfile:    strings.TrimSpace(input.PublicProfile),
			PublicGoalsJson:  mustJSON(input.PublicGoals),
			AttributesJson:   rawJSON(input.Attributes),
			PrivatePayloadID: sql.NullString{},
			VisibilityScope:  defaultVisibility(input.Visibility, "role"),
			Sensitivity:      sensitivityNumber(defaultString(input.Sensitivity, "D2")),
			SourceRefsJson:   optionalRawJSONRaw(input.SourceRefs),
			GeneratedOrigin:  defaultString(input.GeneratedOrigin, "source"),
			LockedFieldsJson: lockedFieldsRaw(input.LockedFields),
			SortOrder:        uint32(input.SortOrder),
		}
		if params.CharacterKey == "" || params.DisplayName == "" || params.PublicProfile == "" {
			return "", nil, validationError(
				"character_key, display_name, and public_profile are required",
			)
		}
		if !oneOf(params.RoleClass, "player_role", "npc", "faction_leader") {
			return "", nil, validationError("role_class is invalid")
		}
		if !oneOf(params.GeneratedOrigin, "source", "confirmed", "inferred", "pending") {
			return "", nil, validationError("generated_origin is invalid")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if err := validateSensitivity(input.Sensitivity); err != nil {
			return "", nil, err
		}
		if update {
			if err := repo.UpdateScriptCharacter(ctx, sqlc.UpdateScriptCharacterParams{
				CharacterKey:     params.CharacterKey,
				DisplayName:      params.DisplayName,
				RoleClass:        params.RoleClass,
				PublicProfile:    params.PublicProfile,
				PublicGoalsJson:  params.PublicGoalsJson,
				AttributesJson:   params.AttributesJson,
				PrivatePayloadID: params.PrivatePayloadID,
				VisibilityScope:  params.VisibilityScope,
				Sensitivity:      params.Sensitivity,
				SourceRefsJson:   params.SourceRefsJson,
				GeneratedOrigin:  params.GeneratedOrigin,
				LockedFieldsJson: params.LockedFieldsJson,
				SortOrder:        params.SortOrder,
				ID:               newID,
				ScriptVersionID:  versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptCharacter(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptCharacter(ctx, sqlc.GetScriptCharacterParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, characterFromRow(row), nil
	case "clue":
		var input ClueInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if input.PrivateTextRef != nil {
			return "", nil, apperror.New(
				"VALIDATION_ERROR",
				"private_text_ref cannot be written until SecretPayload AAD is approved.",
				false,
				nil,
			)
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptClueParams{
			ID:                 newID,
			ScriptVersionID:    versionID,
			ClueKey:            strings.TrimSpace(input.ClueKey),
			Title:              strings.TrimSpace(input.Title),
			PublicText:         nullableStringParam(input.PublicText),
			ClueType:           strings.TrimSpace(input.ClueType),
			IsRepeatable:       input.IsRepeatable,
			MaxDiscoveries:     nullableInt32Param(input.MaxDiscoveries),
			VisibilityScope:    defaultVisibility(input.Visibility, "public"),
			Sensitivity:        sensitivityNumber(defaultString(input.Sensitivity, "D1")),
			RevealGateID:       nullableStringParam(input.RevealGateID),
			PrivatePayloadID:   sql.NullString{},
			RelatedFactIdsJson: optionalRawJSON(input.RelatedFactIDs),
			SourceRefsJson:     optionalRawJSONRaw(input.SourceRefs),
			SortOrder:          uint32(input.SortOrder),
		}
		if params.ClueKey == "" || params.Title == "" || params.ClueType == "" {
			return "", nil, validationError("clue_key, title, and clue_type are required")
		}
		if !oneOf(params.ClueType, "public", "private", "physical", "testimony") {
			return "", nil, validationError("clue_type is invalid")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if err := validateSensitivity(input.Sensitivity); err != nil {
			return "", nil, err
		}
		if params.RevealGateID.Valid && id.Validate(params.RevealGateID.String) != nil {
			return "", nil, validationError("reveal_gate_id must be a ULID")
		}
		if update {
			if err := repo.UpdateScriptClue(ctx, sqlc.UpdateScriptClueParams{
				ClueKey:            params.ClueKey,
				Title:              params.Title,
				PublicText:         params.PublicText,
				ClueType:           params.ClueType,
				IsRepeatable:       params.IsRepeatable,
				MaxDiscoveries:     params.MaxDiscoveries,
				VisibilityScope:    params.VisibilityScope,
				Sensitivity:        params.Sensitivity,
				RevealGateID:       params.RevealGateID,
				PrivatePayloadID:   params.PrivatePayloadID,
				RelatedFactIdsJson: params.RelatedFactIdsJson,
				SourceRefsJson:     params.SourceRefsJson,
				SortOrder:          params.SortOrder,
				ID:                 newID,
				ScriptVersionID:    versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptClue(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptClue(ctx, sqlc.GetScriptClueParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, clueFromRow(row), nil
	case "ending":
		var input EndingInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptEndingParams{
			ID:              newID,
			ScriptVersionID: versionID,
			EndingKey:       strings.TrimSpace(input.EndingKey),
			Title:           strings.TrimSpace(input.Title),
			Description:     strings.TrimSpace(input.Description),
			ConditionsJson:  mustJSON(input.Conditions),
			IsHidden:        input.IsHidden,
			VisibilityScope: defaultVisibility(input.Visibility, "room"),
			SortOrder:       uint32(input.SortOrder),
		}
		if params.EndingKey == "" || params.Title == "" || params.Description == "" {
			return "", nil, validationError("ending_key, title, and description are required")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if update {
			if err := repo.UpdateScriptEnding(ctx, sqlc.UpdateScriptEndingParams{
				EndingKey:       params.EndingKey,
				Title:           params.Title,
				Description:     params.Description,
				ConditionsJson:  params.ConditionsJson,
				IsHidden:        params.IsHidden,
				VisibilityScope: params.VisibilityScope,
				SortOrder:       params.SortOrder,
				ID:              newID,
				ScriptVersionID: versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptEnding(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptEnding(ctx, sqlc.GetScriptEndingParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, endingFromRow(row), nil
	case "asset":
		var input AssetInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptAssetParams{
			ID:              newID,
			ScriptVersionID: versionID,
			AssetKey:        strings.TrimSpace(input.AssetKey),
			AssetType:       strings.TrimSpace(input.AssetType),
			Uri:             strings.TrimSpace(input.URI),
			MimeType:        nullableStringParam(input.MIMEType),
			Sha256:          nullableStringParam(input.SHA256),
			ByteSize:        nullableInt64Param(input.ByteSize),
			VisibilityScope: defaultVisibility(input.Visibility, "public"),
			Sensitivity:     sensitivityNumber(defaultString(input.Sensitivity, "D0")),
			MetadataJson:    optionalRawJSONRaw(input.Metadata),
			SortOrder:       uint32(input.SortOrder),
		}
		if params.AssetKey == "" || params.AssetType == "" || params.Uri == "" {
			return "", nil, validationError("asset_key, asset_type, and uri are required")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if err := validateSensitivity(input.Sensitivity); err != nil {
			return "", nil, err
		}
		if params.Sha256.Valid && !sha256Pattern.MatchString(params.Sha256.String) {
			return "", nil, validationError("sha256 must be 64 lowercase hexadecimal characters")
		}
		if err := validateAssetURI(params.Uri); err != nil {
			return "", nil, validationError(err.Error())
		}
		if update {
			if err := repo.UpdateScriptAsset(ctx, sqlc.UpdateScriptAssetParams{
				AssetKey:        params.AssetKey,
				AssetType:       params.AssetType,
				Uri:             params.Uri,
				MimeType:        params.MimeType,
				Sha256:          params.Sha256,
				ByteSize:        params.ByteSize,
				VisibilityScope: params.VisibilityScope,
				Sensitivity:     params.Sensitivity,
				MetadataJson:    params.MetadataJson,
				SortOrder:       params.SortOrder,
				ID:              newID,
				ScriptVersionID: versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptAsset(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptAsset(ctx, sqlc.GetScriptAssetParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, assetFromRow(row), nil
	case "search_point":
		var input SearchPointInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if err := validateObjectIdentity(update, objectID); err != nil {
			return "", nil, err
		}
		newID := objectID
		if !update {
			newID = s.ids.New()
		}
		params := sqlc.CreateScriptSearchPointParams{
			ID:                newID,
			ScriptVersionID:   versionID,
			PointKey:          strings.TrimSpace(input.PointKey),
			PhaseID:           strings.TrimSpace(input.StageID),
			SceneID:           nullableStringParam(input.SceneID),
			Title:             strings.TrimSpace(input.Title),
			PublicDescription: strings.TrimSpace(input.PublicDescription),
			MaxSearches:       uint32(input.MaxSearches),
			ConflictPolicy:    strings.TrimSpace(input.ConflictPolicy),
			VisibilityScope:   defaultVisibility(input.Visibility, "room"),
			RevealGateID:      nullableStringParam(input.RevealGateID),
			MetadataJson:      optionalRawJSONRaw(input.Metadata),
		}
		if params.PointKey == "" || params.PhaseID == "" || params.Title == "" {
			return "", nil, validationError("point_key, stage_id, and title are required")
		}
		if !oneOf(params.ConflictPolicy, "first_wins", "repeatable", "queue") {
			return "", nil, validationError("conflict_policy is invalid")
		}
		if err := validateVisibility(params.VisibilityScope); err != nil {
			return "", nil, err
		}
		if id.Validate(params.PhaseID) != nil {
			return "", nil, validationError("stage_id must be a ULID")
		}
		if params.SceneID.Valid && id.Validate(params.SceneID.String) != nil {
			return "", nil, validationError("scene_id must be a ULID")
		}
		if params.RevealGateID.Valid && id.Validate(params.RevealGateID.String) != nil {
			return "", nil, validationError("reveal_gate_id must be a ULID")
		}
		if update {
			if err := repo.UpdateScriptSearchPoint(ctx, sqlc.UpdateScriptSearchPointParams{
				PointKey:          params.PointKey,
				PhaseID:           params.PhaseID,
				SceneID:           params.SceneID,
				Title:             params.Title,
				PublicDescription: params.PublicDescription,
				MaxSearches:       params.MaxSearches,
				ConflictPolicy:    params.ConflictPolicy,
				VisibilityScope:   params.VisibilityScope,
				RevealGateID:      params.RevealGateID,
				MetadataJson:      params.MetadataJson,
				ID:                newID,
				ScriptVersionID:   versionID,
			}); err != nil {
				return "", nil, mapObjectStorageError(err)
			}
		} else if err := repo.CreateScriptSearchPoint(ctx, params); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		row, err := repo.GetScriptSearchPoint(ctx, sqlc.GetScriptSearchPointParams{
			ID: newID, ScriptVersionID: versionID,
		})
		if err != nil {
			return "", nil, objectNotFoundError()
		}
		return newID, searchPointFromRow(row), nil
	case "search_point_clue":
		var input SearchPointClueInput
		if err := decodeObject(raw, &input); err != nil {
			return "", nil, err
		}
		if id.Validate(input.SearchPointID) != nil || id.Validate(input.ClueID) != nil {
			return "", nil, validationError("search_point_id and clue_id must be ULIDs")
		}
		if input.Weight < 1 {
			input.Weight = 1
		}
		if err := repo.UpsertScriptSearchPointClue(
			ctx,
			sqlc.UpsertScriptSearchPointClueParams{
				ScriptVersionID: versionID,
				SearchPointID:   input.SearchPointID,
				ClueID:          input.ClueID,
				Weight:          uint32(input.Weight),
				IsDefault:       input.IsDefault,
				MetadataJson:    optionalRawJSONRaw(input.Metadata),
			},
		); err != nil {
			return "", nil, mapObjectStorageError(err)
		}
		rows, err := repo.ListScriptSearchPointClues(ctx, versionID)
		if err != nil {
			return "", nil, mapStorageError(err)
		}
		for _, row := range rows {
			if row.SearchPointID == input.SearchPointID && row.ClueID == input.ClueID {
				return input.SearchPointID + ":" + input.ClueID,
					searchPointClueFromRow(row), nil
			}
		}
		return "", nil, objectNotFoundError()
	default:
		return "", nil, unsupportedObjectTypeError(objectType)
	}
}

func (s *Service) deleteObjectRow(
	ctx context.Context,
	repo storage.Repository,
	versionID string,
	objectType string,
	objectID string,
) error {
	if err := id.Validate(objectID); err != nil {
		return objectNotFoundError()
	}
	var err error
	switch objectType {
	case "chapter":
		err = repo.DeleteScriptChapter(ctx, sqlc.DeleteScriptChapterParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "stage":
		err = repo.DeleteScriptStage(ctx, sqlc.DeleteScriptStageParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "character":
		err = repo.DeleteScriptCharacter(ctx, sqlc.DeleteScriptCharacterParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "clue":
		err = repo.DeleteScriptClue(ctx, sqlc.DeleteScriptClueParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "ending":
		err = repo.DeleteScriptEnding(ctx, sqlc.DeleteScriptEndingParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "asset":
		err = repo.DeleteScriptAsset(ctx, sqlc.DeleteScriptAssetParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	case "search_point":
		err = repo.DeleteScriptSearchPoint(ctx, sqlc.DeleteScriptSearchPointParams{
			ID: objectID, ScriptVersionID: versionID,
		})
	default:
		return unsupportedObjectTypeError(objectType)
	}
	if err != nil {
		return mapObjectStorageError(err)
	}
	return nil
}

func decodeObject(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return apperror.New(
			"VALIDATION_ERROR",
			"The structured object is invalid.",
			false,
			map[string]string{"reason": err.Error()},
		)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return validationError("The structured object must contain one JSON value.")
	}
	return nil
}

func validateObjectIdentity(update bool, objectID string) error {
	if update && id.Validate(objectID) != nil {
		return objectNotFoundError()
	}
	if !update && objectID != "" {
		return validationError("object_id is not accepted when creating an object")
	}
	return nil
}

func rawMessagePointerInput(value *json.RawMessage) *json.RawMessage {
	if value == nil {
		return nil
	}
	copyValue := append(json.RawMessage(nil), (*value)...)
	return &copyValue
}

func defaultVisibility(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func validateVisibility(value string) error {
	if !oneOf(value, "public", "room", "role", "role_group", "player", "runtime", "audit") {
		return validationError("visibility is invalid")
	}
	return nil
}

func validateSensitivity(value string) error {
	if !oneOf(value, "", "D0", "D1", "D2", "D3", "D4") {
		return validationError("sensitivity is invalid")
	}
	return nil
}

func unsupportedObjectTypeError(value string) error {
	return apperror.New(
		"UNSUPPORTED_OBJECT_TYPE",
		"The script object type is not supported.",
		false,
		map[string]string{"object_type": value},
	)
}

func objectNotFoundError() error {
	return apperror.New("OBJECT_NOT_FOUND", "The script object does not exist.", false, nil)
}

func mapObjectStorageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, storage.ErrNoRowsAffected) {
		return objectNotFoundError()
	}
	return mapStorageError(err)
}

func objectMutationResponseStatus(created bool) int16 {
	if created {
		return 201
	}
	return 200
}
