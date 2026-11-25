package script

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage/sqlc"
)

type Service struct {
	repo      storage.Repository
	ids       id.Generator
	loader    *Loader
	validator *Validator
	now       func() time.Time
}

func NewService(repo storage.Repository) *Service {
	return &Service{
		repo:      repo,
		ids:       id.ULIDGenerator{},
		loader:    NewLoader(),
		validator: NewValidator(),
		now:       time.Now,
	}
}

func (s *Service) Create(
	ctx context.Context,
	ownerUserID string,
	input CreateScriptInput,
	idempotencyKey string,
) (CreateScriptResult, error) {
	if idempotencyKey == "" {
		return CreateScriptResult{}, validationError("Idempotency-Key is required")
	}
	normalized, err := validateCreateInput(input)
	if err != nil {
		return CreateScriptResult{}, err
	}
	requestHash := hashJSON(normalized)
	var result CreateScriptResult
	err = s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.create",
			"user:"+ownerUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		if replay && record.ResourceID.Valid {
			scriptRow, err := tx.GetScript(ctx, record.ResourceID.String)
			if err != nil {
				return scriptNotFoundError()
			}
			versionRow, err := tx.GetScriptVersion(ctx, scriptRow.ActiveDraftVersionID.String)
			if err != nil {
				return versionNotFoundError()
			}
			result = CreateScriptResult{
				Script:           scriptFromRow(scriptRow),
				Version:          versionFromRow(versionRow),
				IdempotentReplay: true,
			}
			return nil
		}
		if normalized.RuleTemplateID != nil {
			if _, err := tx.GetActiveRuleTemplateForScript(
				ctx,
				sqlc.GetActiveRuleTemplateForScriptParams{
					ID:       *normalized.RuleTemplateID,
					GameType: normalized.GameType,
				},
			); err != nil {
				return apperror.New(
					"VALIDATION_ERROR",
					"rule_template_id must reference an active template for this game type.",
					false,
					nil,
				)
			}
		}

		scriptID := s.ids.New()
		versionID := s.ids.New()
		scriptParams := sqlc.CreateScriptParams{
			ID:                   scriptID,
			OwnerUserID:          ownerUserID,
			Title:                normalized.Title,
			Summary:              sql.NullString{},
			GameType:             normalized.GameType,
			SourceKind:           "original",
			RightsType:           normalized.RightsType,
			Status:               "draft",
			CoverAssetUri:        sql.NullString{},
			Language:             normalized.Language,
			ContentRating:        normalized.ContentRating,
			MinPlayers:           uint8(normalized.TargetPlayers.Min),
			MaxPlayers:           uint8(normalized.TargetPlayers.Max),
			DurationMinutes:      uint16(normalized.DurationMinutes),
			Difficulty:           normalized.Difficulty,
			AiAssistedRatioBp:    0,
			ActiveDraftVersionID: sql.NullString{String: versionID, Valid: true},
		}
		if err := tx.CreateScript(ctx, scriptParams); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CreateScriptVersion(ctx, sqlc.CreateScriptVersionParams{
			ID:              versionID,
			ScriptID:        scriptID,
			VersionNo:       1,
			VersionKind:     "draft",
			Status:          "draft",
			SchemaVersion:   "1.0.0",
			WorkflowVersion: "manual-v1",
			RuleTemplateID:  nullableStringParam(normalized.RuleTemplateID),
			BlueprintJson:   json.RawMessage(`{"schema_version":"1.0.0"}`),
			CreatedBy:       ownerUserID,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 201, Valid: true},
			ResourceType:   sql.NullString{String: "script", Valid: true},
			ResourceID:     sql.NullString{String: scriptID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		scriptRow, err := tx.GetScript(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		versionRow, err := tx.GetScriptVersion(ctx, versionID)
		if err != nil {
			return versionNotFoundError()
		}
		result = CreateScriptResult{
			Script:  scriptFromRow(scriptRow),
			Version: versionFromRow(versionRow),
		}
		return nil
	})
	return result, err
}

func (s *Service) ListPublic(
	ctx context.Context,
	q string,
	gameType string,
	minPlayers int,
	maxPlayers int,
	durationMax int,
	difficulty string,
	cursor string,
	limit int,
) (CatalogPage, error) {
	if limit < 1 || limit > 100 {
		return CatalogPage{}, validationError("limit must be between 1 and 100")
	}
	rows, err := s.repo.ListPublicScripts(ctx, sqlc.ListPublicScriptsParams{
		Column1:         q,
		CONCAT:          q,
		Column3:         gameType,
		GameType:        gameType,
		Column5:         minPlayers,
		MinPlayers:      uint8(minPlayers),
		Column7:         maxPlayers,
		MaxPlayers:      uint8(maxPlayers),
		Column9:         durationMax,
		DurationMinutes: uint16(durationMax),
		Column11:        difficulty,
		Difficulty:      difficulty,
		Column13:        cursor,
		ID:              cursor,
		Limit:           int32(limit),
	})
	if err != nil {
		return CatalogPage{}, mapStorageError(err)
	}
	items := make([]ScriptCatalogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ScriptCatalogItem{
			ScriptID:        row.ScriptID,
			OwnerUserID:     row.OwnerUserID,
			Title:           row.Title,
			Summary:         nullStringPointer(row.Summary),
			GameType:        row.GameType,
			CoverAssetURI:   nullStringPointer(row.CoverAssetUri),
			Language:        row.Language,
			ContentRating:   row.ContentRating,
			MinPlayers:      int(row.MinPlayers),
			MaxPlayers:      int(row.MaxPlayers),
			DurationMinutes: int(row.DurationMinutes),
			Difficulty:      row.Difficulty,
			ScriptVersionID: row.ScriptVersionID,
			VersionNo:       int(row.VersionNo),
			SchemaVersion:   row.SchemaVersion,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	var next *string
	if len(items) == limit {
		value := items[len(items)-1].ScriptID
		next = &value
	}
	return CatalogPage{Items: items, NextCursor: next}, nil
}

func (s *Service) PublicDetail(ctx context.Context, scriptID string) (PublicScriptDetail, error) {
	if err := id.Validate(scriptID); err != nil {
		return PublicScriptDetail{}, contentNotFoundError()
	}
	row, err := s.repo.GetPublicScriptDetail(ctx, scriptID)
	if err != nil {
		return PublicScriptDetail{}, contentNotFoundError()
	}
	loaded, err := s.loader.LoadPlayable(
		ctx,
		s.repo,
		row.ScriptVersionID,
		LoadModeRuntime,
	)
	if err != nil {
		return PublicScriptDetail{}, err
	}
	content := publicContent(loaded.Content)
	return PublicScriptDetail{
		Script:  content.Script,
		Version: content.Version,
		Content: content,
	}, nil
}

func (s *Service) EditorContent(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
) (Content, error) {
	scriptRow, versionRow, err := s.ownedVersion(ctx, s.repo, actorUserID, scriptID, versionID)
	if err != nil {
		return Content{}, err
	}
	content, err := s.loader.LoadEditor(ctx, s.repo, versionID)
	if err != nil {
		return Content{}, err
	}
	content.Script = scriptFromRow(scriptRow)
	content.Version = versionFromRow(versionRow)
	return content, nil
}

func (s *Service) UpdateMetadata(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	input UpdateScriptInput,
	idempotencyKey string,
) (ScriptDetail, error) {
	if idempotencyKey == "" {
		return ScriptDetail{}, validationError("Idempotency-Key is required")
	}
	if input.RowVersion == 0 {
		return ScriptDetail{}, validationError("row_version is required")
	}
	var result ScriptDetail
	requestHash := hashJSON(input)
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.update",
			"user:"+actorUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		scriptRow, err := tx.GetScriptForUpdate(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		if scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		if replay {
			result, err = s.scriptDetail(ctx, tx, scriptRow)
			return err
		}
		if err := validateScriptPatch(input.Patch); err != nil {
			return err
		}
		updated := applyScriptPatch(scriptRow, input.Patch)
		if err := validateScriptMetadata(updated); err != nil {
			return err
		}
		if err := tx.UpdateScriptMetadata(ctx, sqlc.UpdateScriptMetadataParams{
			Title:           updated.Title,
			Summary:         updated.Summary,
			CoverAssetUri:   updated.CoverAssetUri,
			Language:        updated.Language,
			ContentRating:   updated.ContentRating,
			MinPlayers:      updated.MinPlayers,
			MaxPlayers:      updated.MaxPlayers,
			DurationMinutes: updated.DurationMinutes,
			Difficulty:      updated.Difficulty,
			ID:              scriptID,
			OwnerUserID:     actorUserID,
			RowVersion:      input.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script", Valid: true},
			ResourceID:     sql.NullString{String: scriptID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		scriptRow, err = tx.GetScript(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		result, err = s.scriptDetail(ctx, tx, scriptRow)
		return err
	})
	return result, err
}

func (s *Service) SoftDelete(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	input DeleteScriptInput,
	idempotencyKey string,
) (DeleteScriptResult, error) {
	if idempotencyKey == "" {
		return DeleteScriptResult{}, validationError("Idempotency-Key is required")
	}
	var result DeleteScriptResult
	requestHash := hashJSON(input)
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.delete",
			"user:"+actorUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		scriptRow, err := tx.GetScriptAny(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		if scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		if replay {
			result = DeleteScriptResult{
				ScriptID:   scriptRow.ID,
				Status:     scriptRow.Status,
				RowVersion: scriptRow.RowVersion,
			}
			return nil
		}
		if scriptRow.Status == "soft_deleted" {
			return versionFrozenError("The script is already soft-deleted.")
		}
		if scriptRow.PublishedVersionID.Valid {
			return apperror.New(
				"VERSION_FROZEN",
				"Unpublish the script before deleting it.",
				false,
				nil,
			)
		}
		if input.RowVersion != scriptRow.RowVersion {
			return versionConflictError()
		}
		if err := tx.SoftDeleteScript(ctx, sqlc.SoftDeleteScriptParams{
			DeletedAt:   sql.NullTime{Time: s.now().UTC(), Valid: true},
			ID:          scriptID,
			OwnerUserID: actorUserID,
			RowVersion:  input.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script", Valid: true},
			ResourceID:     sql.NullString{String: scriptID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		result = DeleteScriptResult{
			ScriptID:   scriptID,
			Status:     "soft_deleted",
			RowVersion: input.RowVersion + 1,
		}
		return nil
	})
	return result, err
}

func (s *Service) Versions(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	status string,
	limit int,
) ([]VersionSummary, error) {
	scriptRow, err := s.repo.GetScript(ctx, scriptID)
	if err != nil {
		return nil, scriptNotFoundError()
	}
	if scriptRow.OwnerUserID != actorUserID {
		return nil, forbiddenViewError()
	}
	if limit < 1 || limit > 100 {
		return nil, validationError("limit must be between 1 and 100")
	}
	rows, err := s.repo.ListScriptVersions(ctx, sqlc.ListScriptVersionsParams{
		ScriptID: scriptID,
		Column2:  status,
		Status:   status,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, mapStorageError(err)
	}
	items := make([]VersionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, VersionSummary{
			VersionID:          row.ID,
			ScriptID:           row.ScriptID,
			VersionNo:          int(row.VersionNo),
			VersionKind:        row.VersionKind,
			Status:             row.Status,
			SchemaVersion:      row.SchemaVersion,
			ValidationStatus:   row.ValidationStatus,
			BlockingIssueCount: int(row.BlockingIssueCount),
			CreatedAt:          row.CreatedAt,
			FrozenAt:           nullTimePointer(row.FrozenAt),
			PublishedAt:        nullTimePointer(row.PublishedAt),
			RowVersion:         row.RowVersion,
		})
	}
	return items, nil
}

func (s *Service) CreateDraft(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	input CreateDraftInput,
	idempotencyKey string,
) (Content, error) {
	if idempotencyKey == "" {
		return Content{}, validationError("Idempotency-Key is required")
	}
	requestHash := hashJSON(input)
	var result Content
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.version.clone",
			"user:"+actorUserID,
			idempotencyKey,
			requestHash,
		)
		if err != nil {
			return err
		}
		scriptRow, err := tx.GetScriptForUpdate(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		if scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		if replay && record.ResourceID.Valid {
			result, err = s.loader.LoadEditor(ctx, tx, record.ResourceID.String)
			return err
		}
		sourceVersion, err := tx.GetScriptVersion(ctx, versionID)
		if err != nil || sourceVersion.ScriptID != scriptID {
			return versionNotFoundError()
		}
		source, err := s.loader.LoadEditor(ctx, tx, versionID)
		if err != nil {
			return err
		}
		nextVersionNo, err := tx.GetNextScriptVersionNo(ctx, scriptID)
		if err != nil {
			return mapStorageError(err)
		}
		newVersionID := s.ids.New()
		ruleTemplateID := sourceVersion.RuleTemplateID
		if input.RuleTemplateID != nil {
			if _, err := tx.GetActiveRuleTemplateForScript(
				ctx,
				sqlc.GetActiveRuleTemplateForScriptParams{
					ID:       *input.RuleTemplateID,
					GameType: scriptRow.GameType,
				},
			); err != nil {
				return validationError("rule_template_id is not an active template")
			}
			ruleTemplateID = sql.NullString{String: *input.RuleTemplateID, Valid: true}
		}
		if err := tx.CreateScriptVersion(ctx, sqlc.CreateScriptVersionParams{
			ID:              newVersionID,
			ScriptID:        scriptID,
			VersionNo:       uint32(nextVersionNo),
			VersionKind:     "draft",
			Status:          "draft",
			SchemaVersion:   sourceVersion.SchemaVersion,
			WorkflowVersion: sourceVersion.WorkflowVersion,
			BaseVersionID:   sql.NullString{String: versionID, Valid: true},
			RuleTemplateID:  ruleTemplateID,
			BlueprintJson:   append(json.RawMessage(nil), sourceVersion.BlueprintJson...),
			CreatedBy:       actorUserID,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := s.cloneContent(ctx, tx, newVersionID, source); err != nil {
			return err
		}
		if err := tx.SetScriptActiveDraftVersion(
			ctx,
			sqlc.SetScriptActiveDraftVersionParams{
				ActiveDraftVersionID: sql.NullString{String: newVersionID, Valid: true},
				ID:                   scriptID,
				RowVersion:           scriptRow.RowVersion,
			},
		); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 201, Valid: true},
			ResourceType:   sql.NullString{String: "script_version", Valid: true},
			ResourceID:     sql.NullString{String: newVersionID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		result, err = s.loader.LoadEditor(ctx, tx, newVersionID)
		return err
	})
	return result, err
}

func (s *Service) Validate(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	input ValidateInput,
	idempotencyKey string,
) (ValidationReport, error) {
	if idempotencyKey == "" {
		return ValidationReport{}, validationError("Idempotency-Key is required")
	}
	if input.ExpectedRowVersion == 0 {
		return ValidationReport{}, validationError("expected_row_version is required")
	}
	if input.Profile != "" && input.Profile != "publish" && input.Profile != "structure" {
		return ValidationReport{}, validationError("validation_profile is invalid")
	}
	var report ValidationReport
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.version.validate",
			"user:"+actorUserID,
			idempotencyKey,
			hashJSON(input),
		)
		if err != nil {
			return err
		}
		versionRow, err := tx.GetScriptVersionForUpdate(ctx, versionID)
		if err != nil || versionRow.ScriptID != scriptID {
			return versionNotFoundError()
		}
		scriptRow, err := tx.GetScript(ctx, scriptID)
		if err != nil || scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		if !oneOf(versionRow.Status, "draft", "validating", "review") {
			return versionFrozenError("Published script versions cannot be validated as drafts.")
		}
		if replay && versionRow.ValidationReportJson != nil &&
			len(*versionRow.ValidationReportJson) > 0 {
			var stored ValidationReport
			if err := json.Unmarshal(*versionRow.ValidationReportJson, &stored); err == nil {
				report = stored
				report.VersionRowVersion = versionRow.RowVersion
				return nil
			}
		}
		if input.ExpectedRowVersion != 0 && input.ExpectedRowVersion != versionRow.RowVersion {
			return versionConflictError()
		}
		content, err := s.loader.LoadEditor(ctx, tx, versionID)
		if err != nil {
			return err
		}
		report, err = s.validator.Validate(ctx, tx, content)
		if err != nil {
			return err
		}
		reportRaw, _ := json.Marshal(report)
		raw := json.RawMessage(reportRaw)
		if err := tx.UpdateScriptDraftValidation(ctx, sqlc.UpdateScriptDraftValidationParams{
			ValidationStatus:     report.Status,
			BlockingIssueCount:   uint32(report.BlockingCount),
			ValidationReportJson: &raw,
			ID:                   versionID,
			RowVersion:           versionRow.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		report.VersionRowVersion = versionRow.RowVersion + 1
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script_version", Valid: true},
			ResourceID:     sql.NullString{String: versionID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		return nil
	})
	return report, err
}

func (s *Service) Publish(
	ctx context.Context,
	actorUserID string,
	scriptID string,
	versionID string,
	input PublishInput,
	idempotencyKey string,
) (PublishResult, error) {
	if idempotencyKey == "" {
		return PublishResult{}, validationError("Idempotency-Key is required")
	}
	if !oneOf(input.Channel, "private", "invite", "public") {
		return PublishResult{}, validationError("channel must be private, invite, or public")
	}
	if input.ExpectedRowVersion == 0 {
		return PublishResult{}, validationError("expected_row_version is required")
	}
	if input.Channel == "public" {
		return PublishResult{}, apperror.New(
			"MODERATION_REQUIRED",
			"Public publishing requires the approved moderation workflow.",
			false,
			nil,
		)
	}
	if !input.ContentRatingAck {
		return PublishResult{}, validationError("content_rating_confirm must be true")
	}
	var result PublishResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.version.publish",
			"user:"+actorUserID,
			idempotencyKey,
			hashJSON(input),
		)
		if err != nil {
			return err
		}
		scriptRow, err := tx.GetScriptForUpdate(ctx, scriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		if scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		versionRow, err := tx.GetScriptVersionForUpdate(ctx, versionID)
		if err != nil || versionRow.ScriptID != scriptID {
			return versionNotFoundError()
		}
		if replay && record.ResourceID.Valid {
			publication, err := tx.GetScriptPublicationForUpdate(ctx, record.ResourceID.String)
			if err != nil {
				return publicationNotFoundError()
			}
			result = PublishResult{
				Publication: publicationFromRow(publication),
				Version:     versionFromRow(versionRow),
				Replay:      true,
			}
			return nil
		}
		if input.ExpectedRowVersion != 0 && input.ExpectedRowVersion != versionRow.RowVersion {
			return versionConflictError()
		}
		if !oneOf(versionRow.Status, "draft", "validating", "review") {
			return versionFrozenError("The script version is already frozen.")
		}
		content, err := s.loader.LoadEditor(ctx, tx, versionID)
		if err != nil {
			return err
		}
		report, err := s.validator.Validate(ctx, tx, content)
		if err != nil {
			return err
		}
		if report.HasBlockingIssues() {
			return apperror.New(
				"PUBLISH_BLOCKED",
				"The script version has blocking validation issues.",
				false,
				report,
			)
		}
		now := s.now().UTC()
		reportRaw, _ := json.Marshal(report)
		reportJSON := json.RawMessage(reportRaw)
		publishStatus := "published_private"
		if err := tx.PublishScriptVersion(ctx, sqlc.PublishScriptVersionParams{
			Status:               publishStatus,
			ValidationStatus:     report.Status,
			ValidationReportJson: &reportJSON,
			FrozenAt:             sql.NullTime{Time: now, Valid: true},
			PublishedAt:          sql.NullTime{Time: now, Valid: true},
			ID:                   versionID,
			ScriptID:             scriptID,
			RowVersion:           versionRow.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.SupersedeActiveScriptPublications(
			ctx,
			sqlc.SupersedeActiveScriptPublicationsParams{
				UnpublishedAt: sql.NullTime{Time: now, Valid: true},
				ScriptID:      scriptID,
				Channel:       input.Channel,
			},
		); err != nil {
			return mapStorageError(err)
		}
		activeKey := scriptID + ":" + input.Channel
		publicationID := s.ids.New()
		existing, err := tx.GetScriptPublicationByChannel(
			ctx,
			sqlc.GetScriptPublicationByChannelParams{
				ScriptID:        scriptID,
				ScriptVersionID: versionID,
				Channel:         input.Channel,
			},
		)
		if err == nil {
			publicationID = existing.ID
			if err := tx.ReactivateScriptPublication(
				ctx,
				sqlc.ReactivateScriptPublicationParams{
					ActivePublicationKey: sql.NullString{String: activeKey, Valid: true},
					PublishedAt:          now,
					ID:                   publicationID,
				},
			); err != nil {
				return mapStorageError(err)
			}
		} else if isStorageCode(err, database.CodeNotFound) {
			if err := tx.CreateScriptPublication(
				ctx,
				sqlc.CreateScriptPublicationParams{
					ID:                   publicationID,
					ScriptID:             scriptID,
					ScriptVersionID:      versionID,
					Channel:              input.Channel,
					ActivePublicationKey: sql.NullString{String: activeKey, Valid: true},
					SubmittedBy:          actorUserID,
					PublishedAt:          now,
				},
			); err != nil {
				return mapStorageError(err)
			}
		} else {
			return mapStorageError(err)
		}
		if err := tx.SetScriptPublishedVersion(
			ctx,
			sqlc.SetScriptPublishedVersionParams{
				PublishedVersionID: sql.NullString{String: versionID, Valid: true},
				ID:                 scriptID,
				RowVersion:         scriptRow.RowVersion,
			},
		); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script_publication", Valid: true},
			ResourceID:     sql.NullString{String: publicationID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		publication, err := tx.GetScriptPublicationForUpdate(ctx, publicationID)
		if err != nil {
			return publicationNotFoundError()
		}
		versionRow, err = tx.GetScriptVersion(ctx, versionID)
		if err != nil {
			return versionNotFoundError()
		}
		result = PublishResult{
			Publication: publicationFromRow(publication),
			Version:     versionFromRow(versionRow),
		}
		return nil
	})
	return result, err
}

func (s *Service) Unpublish(
	ctx context.Context,
	actorUserID string,
	publicationID string,
	input UnpublishInput,
	idempotencyKey string,
) (UnpublishResult, error) {
	if idempotencyKey == "" {
		return UnpublishResult{}, validationError("Idempotency-Key is required")
	}
	if input.ExpectedStatus != "" && input.ExpectedStatus != "active" {
		return UnpublishResult{}, validationError("expected_status must be active")
	}
	if strings.TrimSpace(input.ReasonCode) == "" {
		return UnpublishResult{}, validationError("reason_code is required")
	}
	var result UnpublishResult
	err := s.repo.WithTransaction(ctx, func(ctx context.Context, tx storage.Repository) error {
		record, replay, err := s.idempotency(
			ctx,
			tx,
			"script.publication.unpublish",
			"user:"+actorUserID,
			idempotencyKey,
			hashJSON(input),
		)
		if err != nil {
			return err
		}
		publication, err := tx.GetScriptPublicationForUpdate(ctx, publicationID)
		if err != nil {
			return publicationNotFoundError()
		}
		scriptRow, err := tx.GetScriptForUpdate(ctx, publication.ScriptID)
		if err != nil {
			return scriptNotFoundError()
		}
		if scriptRow.OwnerUserID != actorUserID {
			return forbiddenEditError()
		}
		versionRow, err := tx.GetScriptVersion(ctx, publication.ScriptVersionID)
		if err != nil {
			return versionNotFoundError()
		}
		if replay || publication.Status == "unpublished" {
			result = UnpublishResult{
				Publication: publicationFromRow(publication),
				Version:     versionFromRow(versionRow),
				Replay:      replay,
			}
			return nil
		}
		now := s.now().UTC()
		if err := tx.UnpublishScriptPublication(
			ctx,
			sqlc.UnpublishScriptPublicationParams{
				UnpublishedAt: sql.NullTime{Time: now, Valid: true},
				ReasonCode:    nullableStringParamValue(input.ReasonCode),
				ReasonText:    nullableStringParamValue(input.Reason),
				ID:            publicationID,
			},
		); err != nil {
			return mapStorageError(err)
		}
		if err := tx.MarkScriptVersionUnlisted(ctx, publication.ScriptVersionID); err != nil {
			return mapStorageError(err)
		}
		if err := tx.SetScriptUnlisted(ctx, sqlc.SetScriptUnlistedParams{
			ID:                 publication.ScriptID,
			PublishedVersionID: sql.NullString{String: publication.ScriptVersionID, Valid: true},
			RowVersion:         scriptRow.RowVersion,
		}); err != nil {
			return mapStorageError(err)
		}
		if err := tx.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
			ResponseStatus: sql.NullInt16{Int16: 200, Valid: true},
			ResourceType:   sql.NullString{String: "script_publication", Valid: true},
			ResourceID:     sql.NullString{String: publicationID, Valid: true},
			ID:             record.ID,
		}); err != nil {
			return mapStorageError(err)
		}
		publication, _ = tx.GetScriptPublicationForUpdate(ctx, publicationID)
		versionRow, _ = tx.GetScriptVersion(ctx, publication.ScriptVersionID)
		result = UnpublishResult{
			Publication: publicationFromRow(publication),
			Version:     versionFromRow(versionRow),
		}
		return nil
	})
	return result, err
}

func (s *Service) cloneContent(
	ctx context.Context,
	repo storage.Repository,
	versionID string,
	source Content,
) error {
	chapterIDs := make(map[string]string, len(source.Chapters))
	for _, chapter := range source.Chapters {
		newID := s.ids.New()
		chapterIDs[chapter.ID] = newID
		if err := repo.CreateScriptChapter(ctx, sqlc.CreateScriptChapterParams{
			ID:              newID,
			ScriptVersionID: versionID,
			ChapterKey:      chapter.ChapterKey,
			Title:           chapter.Title,
			Summary:         nullableStringParamValueValue(chapter.Summary),
			SequenceNo:      uint32(chapter.SequenceNo),
			SourceRefsJson:  rawJSON(chapter.SourceRefs),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	stageIDs := make(map[string]string, len(source.Stages))
	for _, stage := range source.Stages {
		newID := s.ids.New()
		stageIDs[stage.ID] = newID
		var chapterID sql.NullString
		if stage.ChapterID != nil {
			chapterID = sql.NullString{String: chapterIDs[*stage.ChapterID], Valid: true}
		}
		if err := repo.CreateScriptStage(ctx, sqlc.CreateScriptStageParams{
			ID:                 newID,
			ScriptVersionID:    versionID,
			ChapterID:          chapterID,
			PhaseKey:           stage.Key,
			PhaseType:          stage.Type,
			SequenceNo:         uint32(stage.SequenceNo),
			Title:              stage.Title,
			EntryConditionJson: rawMessagePointerValue(stage.EntryConditions),
			ExitConditionJson:  rawMessagePointerValue(stage.ExitConditions),
			TimeoutSeconds:     nullableInt32Param(stage.TimeoutSeconds),
			FallbackAction:     stage.FallbackAction,
			AllowedActionsJson: mustJSON(stage.AllowedActions),
			VisibilityScope:    stage.Visibility,
			IsTerminal:         stage.IsTerminal,
		}); err != nil {
			return mapStorageError(err)
		}
	}
	for _, character := range source.Characters {
		var sourceRefs *json.RawMessage
		if len(character.SourceRefs) > 0 {
			raw := rawJSON(character.SourceRefs)
			sourceRefs = &raw
		}
		if err := repo.CreateScriptCharacter(ctx, sqlc.CreateScriptCharacterParams{
			ID:               s.ids.New(),
			ScriptVersionID:  versionID,
			CharacterKey:     character.CharacterKey,
			DisplayName:      character.DisplayName,
			RoleClass:        character.RoleClass,
			PublicProfile:    character.PublicProfile,
			PublicGoalsJson:  mustJSON(character.PublicGoals),
			AttributesJson:   rawJSON(character.Attributes),
			PrivatePayloadID: sql.NullString{},
			VisibilityScope:  character.Visibility,
			Sensitivity:      sensitivityNumber(character.Sensitivity),
			SourceRefsJson:   sourceRefs,
			GeneratedOrigin:  character.GeneratedOrigin,
			LockedFieldsJson: lockedFieldsRaw(character.LockedFields),
			SortOrder:        uint32(character.SortOrder),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	clueIDs := make(map[string]string, len(source.Clues))
	for _, clue := range source.Clues {
		newID := s.ids.New()
		clueIDs[clue.ID] = newID
		var publicText sql.NullString
		if clue.PublicText != nil {
			publicText = sql.NullString{String: *clue.PublicText, Valid: true}
		}
		if err := repo.CreateScriptClue(ctx, sqlc.CreateScriptClueParams{
			ID:                 newID,
			ScriptVersionID:    versionID,
			ClueKey:            clue.ClueKey,
			Title:              clue.Title,
			PublicText:         publicText,
			ClueType:           clue.ClueType,
			IsRepeatable:       clue.IsRepeatable,
			MaxDiscoveries:     nullableInt32Param(clue.MaxDiscoveries),
			VisibilityScope:    clue.Visibility,
			Sensitivity:        sensitivityNumber(clue.Sensitivity),
			RevealGateID:       sql.NullString{},
			PrivatePayloadID:   sql.NullString{},
			RelatedFactIdsJson: optionalRawJSON(clue.RelatedFactIDs),
			SourceRefsJson:     optionalRawJSONRaw(clue.SourceRefs),
			SortOrder:          uint32(clue.SortOrder),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	for _, ending := range source.Endings {
		if err := repo.CreateScriptEnding(ctx, sqlc.CreateScriptEndingParams{
			ID:              s.ids.New(),
			ScriptVersionID: versionID,
			EndingKey:       ending.EndingKey,
			Title:           ending.Title,
			Description:     ending.Description,
			ConditionsJson:  rawJSON(ending.Conditions),
			IsHidden:        ending.IsHidden,
			VisibilityScope: ending.Visibility,
			SortOrder:       uint32(ending.SortOrder),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	for _, asset := range source.Assets {
		if err := repo.CreateScriptAsset(ctx, sqlc.CreateScriptAssetParams{
			ID:              s.ids.New(),
			ScriptVersionID: versionID,
			AssetKey:        asset.AssetKey,
			AssetType:       asset.AssetType,
			Uri:             asset.URI,
			MimeType:        nullableStringParamValueValue(asset.MIMEType),
			Sha256:          nullableStringParamValueValue(asset.SHA256),
			ByteSize:        nullableInt64Param(asset.ByteSize),
			VisibilityScope: asset.Visibility,
			Sensitivity:     sensitivityNumber(asset.Sensitivity),
			MetadataJson:    optionalRawJSONRaw(asset.Metadata),
			SortOrder:       uint32(asset.SortOrder),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	searchPointIDs := make(map[string]string, len(source.SearchPoints))
	for _, point := range source.SearchPoints {
		newID := s.ids.New()
		searchPointIDs[point.ID] = newID
		if err := repo.CreateScriptSearchPoint(ctx, sqlc.CreateScriptSearchPointParams{
			ID:                newID,
			ScriptVersionID:   versionID,
			PointKey:          point.PointKey,
			PhaseID:           stageIDs[point.StageID],
			SceneID:           sql.NullString{},
			Title:             point.Title,
			PublicDescription: point.PublicDescription,
			MaxSearches:       uint32(point.MaxSearches),
			ConflictPolicy:    point.ConflictPolicy,
			VisibilityScope:   point.Visibility,
			RevealGateID:      sql.NullString{},
			MetadataJson:      optionalRawJSONRaw(point.Metadata),
		}); err != nil {
			return mapStorageError(err)
		}
	}
	for _, link := range source.PointClues {
		if err := repo.UpsertScriptSearchPointClue(
			ctx,
			sqlc.UpsertScriptSearchPointClueParams{
				ScriptVersionID: versionID,
				SearchPointID:   searchPointIDs[link.SearchPointID],
				ClueID:          clueIDs[link.ClueID],
				Weight:          uint32(link.Weight),
				IsDefault:       link.IsDefault,
				MetadataJson:    optionalRawJSONRaw(link.Metadata),
			},
		); err != nil {
			return mapStorageError(err)
		}
	}
	return nil
}

func (s *Service) ownedVersion(
	ctx context.Context,
	repo storage.Repository,
	actorUserID string,
	scriptID string,
	versionID string,
) (sqlc.Script, sqlc.ScriptVersion, error) {
	scriptRow, err := repo.GetScript(ctx, scriptID)
	if err != nil {
		return sqlc.Script{}, sqlc.ScriptVersion{}, scriptNotFoundError()
	}
	if scriptRow.OwnerUserID != actorUserID {
		return sqlc.Script{}, sqlc.ScriptVersion{}, forbiddenViewError()
	}
	versionRow, err := repo.GetScriptVersion(ctx, versionID)
	if err != nil || versionRow.ScriptID != scriptID {
		return sqlc.Script{}, sqlc.ScriptVersion{}, versionNotFoundError()
	}
	return scriptRow, versionRow, nil
}

func (s *Service) scriptDetail(
	ctx context.Context,
	repo storage.Repository,
	scriptRow sqlc.Script,
) (ScriptDetail, error) {
	var versionRow sqlc.ScriptVersion
	var err error
	if scriptRow.ActiveDraftVersionID.Valid {
		versionRow, err = repo.GetScriptVersion(ctx, scriptRow.ActiveDraftVersionID.String)
	} else if scriptRow.PublishedVersionID.Valid {
		versionRow, err = repo.GetScriptVersion(ctx, scriptRow.PublishedVersionID.String)
	} else {
		versions, listErr := repo.ListScriptVersions(ctx, sqlc.ListScriptVersionsParams{
			ScriptID: scriptRow.ID,
			Column2:  "",
			Status:   "",
			Limit:    1,
		})
		if listErr != nil || len(versions) == 0 {
			return ScriptDetail{}, versionNotFoundError()
		}
		versionRow = versions[0]
	}
	if err != nil {
		return ScriptDetail{}, versionNotFoundError()
	}
	return ScriptDetail{
		Script:  scriptFromRow(scriptRow),
		Version: versionFromRow(versionRow),
	}, nil
}

func (s *Service) idempotency(
	ctx context.Context,
	repo storage.Repository,
	scope string,
	actorScope string,
	key string,
	requestHash []byte,
) (sqlc.IdempotencyRecord, bool, error) {
	record, err := repo.GetIdempotencyRecord(ctx, sqlc.GetIdempotencyRecordParams{
		Scope:          scope,
		ActorScope:     actorScope,
		IdempotencyKey: key,
	})
	if err == nil {
		if !hmac.Equal(record.RequestHash, requestHash) {
			return sqlc.IdempotencyRecord{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The idempotency key was reused with a different request.",
				false,
				nil,
			)
		}
		if record.Status != "completed" {
			return sqlc.IdempotencyRecord{}, false, apperror.New(
				"IDEMPOTENCY_CONFLICT",
				"The previous request is still in progress.",
				true,
				nil,
			)
		}
		return record, true, nil
	}
	if !isStorageCode(err, database.CodeNotFound) {
		return sqlc.IdempotencyRecord{}, false, mapStorageError(err)
	}
	recordID := s.ids.New()
	if err := repo.CreateIdempotencyRecord(ctx, sqlc.CreateIdempotencyRecordParams{
		ID:             recordID,
		Scope:          scope,
		ActorScope:     actorScope,
		IdempotencyKey: key,
		RequestHash:    requestHash,
		ExpiresAt:      s.now().UTC().Add(24 * time.Hour),
	}); err != nil {
		return sqlc.IdempotencyRecord{}, false, mapStorageError(err)
	}
	return sqlc.IdempotencyRecord{ID: recordID}, false, nil
}

func validateCreateInput(input CreateScriptInput) (CreateScriptInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.GameType = strings.TrimSpace(input.GameType)
	input.RightsType = strings.TrimSpace(input.RightsType)
	input.Language = strings.TrimSpace(input.Language)
	input.Difficulty = strings.TrimSpace(input.Difficulty)
	input.ContentRating = strings.TrimSpace(input.ContentRating)
	if input.ContentRating == "" {
		input.ContentRating = "all"
	}
	if input.RuleTemplateID != nil {
		value := strings.TrimSpace(*input.RuleTemplateID)
		if err := id.Validate(value); err != nil {
			return CreateScriptInput{}, validationError("rule_template_id must be a ULID")
		}
		input.RuleTemplateID = &value
	}
	if utf8.RuneCountInString(input.Title) < 1 || utf8.RuneCountInString(input.Title) > 160 {
		return CreateScriptInput{}, validationError("title must contain between 1 and 160 characters")
	}
	if input.GameType != "script_murder" {
		return CreateScriptInput{}, validationError("game_type must be script_murder")
	}
	if !oneOf(input.RightsType, "original", "authorized", "public_domain", "private_only") {
		return CreateScriptInput{}, validationError("rights_type is invalid")
	}
	if utf8.RuneCountInString(input.Language) < 2 {
		return CreateScriptInput{}, validationError("language is invalid")
	}
	if input.TargetPlayers.Min < 1 ||
		input.TargetPlayers.Max < input.TargetPlayers.Min ||
		input.TargetPlayers.Max > 100 {
		return CreateScriptInput{}, validationError("target_players is invalid")
	}
	if input.DurationMinutes < 1 || input.DurationMinutes > 1440 {
		return CreateScriptInput{}, validationError("duration_minutes is invalid")
	}
	if input.Difficulty == "" {
		return CreateScriptInput{}, validationError("difficulty is required")
	}
	if !oneOf(input.ContentRating, "all", "12+", "16+", "18+") {
		return CreateScriptInput{}, validationError("content_rating is invalid")
	}
	return input, nil
}

func validateScriptMetadata(row sqlc.Script) error {
	if strings.TrimSpace(row.Title) == "" ||
		utf8.RuneCountInString(row.Title) > 160 ||
		utf8.RuneCountInString(row.Language) < 2 ||
		row.MinPlayers < 1 ||
		row.MaxPlayers < row.MinPlayers ||
		row.MaxPlayers > 100 ||
		row.DurationMinutes < 1 ||
		row.DurationMinutes > 1440 ||
		strings.TrimSpace(row.Difficulty) == "" {
		return validationError("script metadata is invalid")
	}
	if !oneOf(row.ContentRating, "all", "12+", "16+", "18+") {
		return validationError("content_rating is invalid")
	}
	return nil
}

func validateScriptPatch(patch UpdateScriptPatch) error {
	if patch.MinPlayers != nil && (*patch.MinPlayers < 1 || *patch.MinPlayers > 100) {
		return validationError("min_players must be between 1 and 100")
	}
	if patch.MaxPlayers != nil && (*patch.MaxPlayers < 1 || *patch.MaxPlayers > 100) {
		return validationError("max_players must be between 1 and 100")
	}
	if patch.DurationMinutes != nil &&
		(*patch.DurationMinutes < 1 || *patch.DurationMinutes > 1440) {
		return validationError("duration_minutes must be between 1 and 1440")
	}
	if patch.Title != nil && utf8.RuneCountInString(strings.TrimSpace(*patch.Title)) > 160 {
		return validationError("title exceeds 160 characters")
	}
	return nil
}

func applyScriptPatch(row sqlc.Script, patch UpdateScriptPatch) sqlc.Script {
	if patch.Title != nil {
		row.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Summary != nil {
		row.Summary = nullableStringParamValueValue(patch.Summary)
	}
	if patch.CoverAssetURI != nil {
		row.CoverAssetUri = nullableStringParamValueValue(patch.CoverAssetURI)
	}
	if patch.Language != nil {
		row.Language = strings.TrimSpace(*patch.Language)
	}
	if patch.ContentRating != nil {
		row.ContentRating = strings.TrimSpace(*patch.ContentRating)
	}
	if patch.MinPlayers != nil {
		row.MinPlayers = uint8(*patch.MinPlayers)
	}
	if patch.MaxPlayers != nil {
		row.MaxPlayers = uint8(*patch.MaxPlayers)
	}
	if patch.DurationMinutes != nil {
		row.DurationMinutes = uint16(*patch.DurationMinutes)
	}
	if patch.Difficulty != nil {
		row.Difficulty = strings.TrimSpace(*patch.Difficulty)
	}
	return row
}

func publicContent(content Content) Content {
	result := content
	result.Characters = make([]Character, 0, len(content.Characters))
	for _, character := range content.Characters {
		character.PrivateProfileRef = nil
		character.PrivatePayloadID = nil
		result.Characters = append(result.Characters, character)
	}
	result.Clues = make([]Clue, 0, len(content.Clues))
	for _, clue := range content.Clues {
		clue.PrivateTextRef = nil
		clue.PrivatePayloadID = nil
		result.Clues = append(result.Clues, clue)
	}
	result.Endings = make([]Ending, 0, len(content.Endings))
	for _, ending := range content.Endings {
		if ending.IsHidden || ending.Visibility != "public" {
			continue
		}
		result.Endings = append(result.Endings, ending)
	}
	result.Assets = make([]Asset, 0, len(content.Assets))
	for _, asset := range content.Assets {
		if asset.Visibility == "public" {
			result.Assets = append(result.Assets, asset)
		}
	}
	return result
}

func publicationFromRow(row sqlc.ScriptPublication) Publication {
	return Publication{
		ID:            row.ID,
		ScriptID:      row.ScriptID,
		VersionID:     row.ScriptVersionID,
		Channel:       row.Channel,
		Status:        row.Status,
		PublishedAt:   row.PublishedAt,
		UnpublishedAt: nullTimePointer(row.UnpublishedAt),
		RowVersion:    row.RowVersion,
	}
}

func mapStorageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, storage.ErrNoRowsAffected) {
		return versionConflictError()
	}
	var databaseError *database.Error
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case database.CodeConflict:
			return versionConflictError()
		case database.CodeNotFound:
			return scriptNotFoundError()
		case database.CodeDependencyUnavailable, database.CodeDeadlineExceeded:
			return apperror.New(
				"DEPENDENCY_ERROR",
				"The content service is temporarily unavailable.",
				true,
				nil,
			)
		}
	}
	return apperror.New("INTERNAL_ERROR", "The script operation failed.", true, nil)
}

func isStorageCode(err error, code string) bool {
	var storageError *database.Error
	return errors.As(err, &storageError) && storageError.Code == code
}

func validationError(message string) error {
	return apperror.New("VALIDATION_ERROR", message, false, nil)
}

func scriptNotFoundError() error {
	return apperror.New("SCRIPT_NOT_FOUND", "The script does not exist.", false, nil)
}

func versionNotFoundError() error {
	return apperror.New("VERSION_NOT_FOUND", "The script version does not exist.", false, nil)
}

func publicationNotFoundError() error {
	return apperror.New("PUBLICATION_NOT_FOUND", "The publication does not exist.", false, nil)
}

func contentNotFoundError() error {
	return apperror.New("CONTENT_NOT_FOUND", "The script content is unavailable.", false, nil)
}

func forbiddenViewError() error {
	return apperror.New("FORBIDDEN_VIEW", "The content is not visible to this user.", false, nil)
}

func forbiddenEditError() error {
	return apperror.New("FORBIDDEN_EDIT", "Only the script owner can edit this content.", false, nil)
}

func versionConflictError() error {
	return apperror.New(
		"VERSION_CONFLICT",
		"The script changed. Refresh and retry.",
		true,
		nil,
	)
}

func versionFrozenError(message string) error {
	return apperror.New("VERSION_FROZEN", message, false, nil)
}

func nullableStringParam(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableStringParamValue(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableStringParamValueValue(value *string) sql.NullString {
	return nullableStringParam(value)
}

func nullableInt32Param(value *int) sql.NullInt32 {
	if value == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*value), Valid: true}
}

func nullableInt64Param(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	copyValue := value.Time
	return &copyValue
}

func optionalRawJSON(values []string) *json.RawMessage {
	if values == nil {
		raw := json.RawMessage("[]")
		return &raw
	}
	raw := mustJSON(values)
	return &raw
}

func optionalRawJSONRaw(value json.RawMessage) *json.RawMessage {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	raw := rawJSON(value)
	return &raw
}

func lockedFieldsRaw(values []string) *json.RawMessage {
	if len(values) == 0 {
		return nil
	}
	raw := mustJSON(values)
	return &raw
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func hashJSON(value any) []byte {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return sum[:]
}
