package script

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/id"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

func TestScriptDraftValidationPublishCloneAndRuntimeLoading(t *testing.T) {
	repo, db := openScriptIntegrationRepository(t)
	ctx := context.Background()
	ids := id.ULIDGenerator{}
	ownerID := ids.New()
	otherID := ids.New()
	seedScriptUser(t, db, ownerID, "ScriptOwner")
	seedScriptUser(t, db, otherID, "OtherUser")
	ruleID := seedMurderRuleTemplate(t, db, ownerID)
	t.Cleanup(func() {
		cleanupScriptTestData(t, db, ownerID, otherID, ruleID)
	})

	service := NewService(repo)
	created, err := service.Create(ctx, ownerID, CreateScriptInput{
		Title:           "Task 7 Integration",
		GameType:        "script_murder",
		RightsType:      "original",
		Language:        "zh-CN",
		TargetPlayers:   TargetPlayers{Min: 2, Max: 2},
		DurationMinutes: 90,
		Difficulty:      "standard",
		ContentRating:   "16+",
		RuleTemplateID:  &ruleID,
	}, ids.New())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	scriptID := created.Script.ID
	versionID := created.Version.ID
	rowVersion := created.Version.RowVersion
	if _, err := NewLoader().LoadPlayable(ctx, repo, versionID, LoadModeStart); err == nil {
		t.Fatal("LoadPlayable(start, draft) unexpectedly succeeded")
	}

	chapter := mustCreateObject(t, service, ownerID, scriptID, versionID, "chapter", rowVersion, ChapterInput{
		ChapterKey: "chapter-one",
		Title:      "Arrival",
		SequenceNo: 1,
		SourceRefs: json.RawMessage("[]"),
	})
	rowVersion = chapter.VersionRowVersion
	chapterID := chapter.ObjectID

	readStage := mustCreateObject(t, service, ownerID, scriptID, versionID, "stage", rowVersion, StageInput{
		ChapterID:      &chapterID,
		StageKey:       "read",
		PhaseType:      "read",
		SequenceNo:     1,
		Title:          "Read",
		AllowedActions: []string{"reading_complete"},
		FallbackAction: "advance_to_search",
		Visibility:     "room",
	})
	rowVersion = readStage.VersionRowVersion
	searchStage := mustCreateObject(t, service, ownerID, scriptID, versionID, "stage", rowVersion, StageInput{
		ChapterID:      &chapterID,
		StageKey:       "search",
		PhaseType:      "search",
		SequenceNo:     2,
		Title:          "Search",
		AllowedActions: []string{"search", "advance_phase"},
		FallbackAction: "advance_to_vote",
		Visibility:     "room",
	})
	rowVersion = searchStage.VersionRowVersion
	voteStage := mustCreateObject(t, service, ownerID, scriptID, versionID, "stage", rowVersion, StageInput{
		ChapterID:      &chapterID,
		StageKey:       "vote",
		PhaseType:      "vote",
		SequenceNo:     3,
		Title:          "Vote",
		AllowedActions: []string{"vote"},
		FallbackAction: "settle",
		Visibility:     "room",
	})
	rowVersion = voteStage.VersionRowVersion
	searchStageID := searchStage.ObjectID

	characterKeys := []string{"killer", "civilian"}
	for index, key := range characterKeys {
		result := mustCreateObject(
			t,
			service,
			ownerID,
			scriptID,
			versionID,
			"character",
			rowVersion,
			CharacterInput{
				CharacterKey:    key,
				DisplayName:     strings.ToUpper(key),
				RoleClass:       "player_role",
				PublicProfile:   "A player character.",
				PublicGoals:     []string{"Find the truth."},
				Attributes:      json.RawMessage("{}"),
				Visibility:      "role",
				Sensitivity:     "D2",
				SourceRefs:      json.RawMessage("[]"),
				GeneratedOrigin: "source",
				SortOrder:       index + 1,
			},
		)
		rowVersion = result.VersionRowVersion
	}

	clue := mustCreateObject(t, service, ownerID, scriptID, versionID, "clue", rowVersion, ClueInput{
		ClueKey:        "torn-ticket",
		Title:          "Torn Ticket",
		PublicText:     stringPointer("A torn ticket."),
		ClueType:       "physical",
		IsRepeatable:   false,
		MaxDiscoveries: intPointer(1),
		Visibility:     "role",
		Sensitivity:    "D1",
		RelatedFactIDs: []string{},
		SourceRefs:     json.RawMessage("[]"),
		SortOrder:      1,
	})
	rowVersion = clue.VersionRowVersion
	clueID := clue.ObjectID

	searchPoint := mustCreateObject(
		t,
		service,
		ownerID,
		scriptID,
		versionID,
		"search_point",
		rowVersion,
		SearchPointInput{
			PointKey:          "desk",
			StageID:           searchStageID,
			Title:             "Desk",
			PublicDescription: "An old desk.",
			MaxSearches:       2,
			ConflictPolicy:    "first_wins",
			Visibility:        "room",
			Metadata:          json.RawMessage("{}"),
		},
	)
	rowVersion = searchPoint.VersionRowVersion
	searchPointID := searchPoint.ObjectID

	link := mustCreateObject(
		t,
		service,
		ownerID,
		scriptID,
		versionID,
		"search_point_clue",
		rowVersion,
		SearchPointClueInput{
			SearchPointID: searchPointID,
			ClueID:        clueID,
			Weight:        1,
			IsDefault:     true,
			Metadata:      json.RawMessage("{}"),
		},
	)
	rowVersion = link.VersionRowVersion
	ending := mustCreateObject(t, service, ownerID, scriptID, versionID, "ending", rowVersion, EndingInput{
		EndingKey:   "solved",
		Title:       "Solved",
		Description: "The killer is identified.",
		Conditions:  []string{},
		IsHidden:    false,
		Visibility:  "room",
		SortOrder:   1,
	})
	rowVersion = ending.VersionRowVersion

	concurrentInput := ObjectMutationInput{
		ExpectedRowVersion: rowVersion,
		Object: mustJSON(ChapterInput{
			ChapterKey: "chapter-one",
			Title:      "Concurrent Revision",
			SequenceNo: 1,
			SourceRefs: json.RawMessage("[]"),
		}),
	}
	type concurrentResult struct {
		result ObjectMutationResult
		err    error
	}
	concurrentResults := make(chan concurrentResult, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := service.UpdateObject(
				ctx,
				ownerID,
				scriptID,
				versionID,
				"chapter",
				chapterID,
				concurrentInput,
				ids.New(),
			)
			concurrentResults <- concurrentResult{result: result, err: err}
		}()
	}
	wait.Wait()
	close(concurrentResults)
	successCount := 0
	conflictCount := 0
	for outcome := range concurrentResults {
		switch {
		case outcome.err == nil:
			successCount++
			rowVersion = outcome.result.VersionRowVersion
		case hasScriptCode(outcome.err, "VERSION_CONFLICT"):
			conflictCount++
		default:
			t.Fatalf("concurrent UpdateObject() error = %v", outcome.err)
		}
	}
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf(
			"concurrent UpdateObject() = %d success, %d conflict",
			successCount,
			conflictCount,
		)
	}

	report, err := service.Validate(
		ctx,
		ownerID,
		scriptID,
		versionID,
		ValidateInput{ExpectedRowVersion: rowVersion},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if report.Status == "blocked" || report.BlockingCount != 0 {
		t.Fatalf("Validate() = %#v", report)
	}
	rowVersion = report.VersionRowVersion
	_, err = service.Publish(
		ctx,
		ownerID,
		scriptID,
		versionID,
		PublishInput{
			Channel:            "public",
			ExpectedRowVersion: rowVersion,
			ContentRatingAck:   true,
		},
		ids.New(),
	)
	if !hasScriptCode(err, "MODERATION_REQUIRED") {
		t.Fatalf("Publish(public) error = %v", err)
	}

	published, err := service.Publish(
		ctx,
		ownerID,
		scriptID,
		versionID,
		PublishInput{
			Channel:            "private",
			ExpectedRowVersion: rowVersion,
			ContentRatingAck:   true,
		},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if published.Version.Status != "published_private" ||
		published.Version.FrozenAt == nil {
		t.Fatalf("published version = %#v", published.Version)
	}
	publishedScript, err := repo.GetScript(ctx, scriptID)
	if err != nil {
		t.Fatalf("GetScript(published) error = %v", err)
	}
	_, err = service.SoftDelete(
		ctx,
		ownerID,
		scriptID,
		DeleteScriptInput{RowVersion: publishedScript.RowVersion, Reason: "test"},
		ids.New(),
	)
	if !hasScriptCode(err, "VERSION_FROZEN") {
		t.Fatalf("SoftDelete(published) error = %v", err)
	}

	_, err = service.UpdateObject(
		ctx,
		ownerID,
		scriptID,
		versionID,
		"chapter",
		chapterID,
		ObjectMutationInput{
			ExpectedRowVersion: published.Version.RowVersion,
			Object: mustJSON(ChapterInput{
				ChapterKey: "chapter-one",
				Title:      "Changed",
				SequenceNo: 1,
				SourceRefs: json.RawMessage("[]"),
			}),
		},
		ids.New(),
	)
	if !hasScriptCode(err, "VERSION_FROZEN") {
		t.Fatalf("UpdateObject(published) error = %v", err)
	}

	clone, err := service.CreateDraft(
		ctx,
		ownerID,
		scriptID,
		versionID,
		CreateDraftInput{DraftName: "Second draft"},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if clone.Version.ID == versionID || clone.Version.BaseVersionID == nil ||
		*clone.Version.BaseVersionID != versionID {
		t.Fatalf("clone version = %#v", clone.Version)
	}
	if len(clone.Characters) != 2 || len(clone.Stages) != 3 {
		t.Fatalf("clone content = %#v", clone)
	}

	cloneChapter := clone.Chapters[0]
	updated, err := service.UpdateObject(
		ctx,
		ownerID,
		scriptID,
		clone.Version.ID,
		"chapter",
		cloneChapter.ID,
		ObjectMutationInput{
			ExpectedRowVersion: clone.Version.RowVersion,
			Object: mustJSON(ChapterInput{
				ChapterKey: cloneChapter.ChapterKey,
				Title:      "Arrival Revised",
				SequenceNo: cloneChapter.SequenceNo,
				Summary:    stringPointer("Revised"),
				SourceRefs: json.RawMessage("[]"),
			}),
		},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("UpdateObject(clone) error = %v", err)
	}
	cloneRowVersion := updated.VersionRowVersion
	cloneReport, err := service.Validate(
		ctx,
		ownerID,
		scriptID,
		clone.Version.ID,
		ValidateInput{ExpectedRowVersion: cloneRowVersion},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Validate(clone) error = %v", err)
	}
	cloneRowVersion = cloneReport.VersionRowVersion
	secondPublish, err := service.Publish(
		ctx,
		ownerID,
		scriptID,
		clone.Version.ID,
		PublishInput{
			Channel:            "private",
			ExpectedRowVersion: cloneRowVersion,
			ContentRatingAck:   true,
		},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Publish(clone) error = %v", err)
	}
	if secondPublish.Publication.VersionID != clone.Version.ID {
		t.Fatalf("second publication = %#v", secondPublish.Publication)
	}

	loader := NewLoader()
	oldRuntime, err := loader.LoadPlayable(ctx, repo, versionID, LoadModeRuntime)
	if err != nil {
		t.Fatalf("LoadPlayable(old runtime) error = %v", err)
	}
	if oldRuntime.Version.ID != versionID || len(oldRuntime.Content.Characters) != 2 {
		t.Fatalf("old runtime content = %#v", oldRuntime)
	}
	if oldRuntime.Content.Characters[0].PrivateProfileRef != nil ||
		oldRuntime.Content.Clues[0].PrivateTextRef != nil {
		t.Fatal("runtime loader exposed private content")
	}

	unpublished, err := service.Unpublish(
		ctx,
		ownerID,
		secondPublish.Publication.ID,
		UnpublishInput{ReasonCode: "test", Reason: "integration"},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Unpublish() error = %v", err)
	}
	if unpublished.Version.Status != "unlisted" {
		t.Fatalf("unpublished version = %#v", unpublished.Version)
	}
	if _, err := loader.LoadPlayable(ctx, repo, clone.Version.ID, LoadModeStart); err == nil {
		t.Fatal("LoadPlayable(start, unlisted) unexpectedly succeeded")
	}
	if _, err := loader.LoadPlayable(ctx, repo, clone.Version.ID, LoadModeRuntime); err != nil {
		t.Fatalf("LoadPlayable(unlisted runtime) error = %v", err)
	}

	_, err = service.EditorContent(ctx, otherID, scriptID, versionID)
	if !hasScriptCode(err, "FORBIDDEN_VIEW") {
		t.Fatalf("EditorContent(non-owner) error = %v", err)
	}
}

func TestUnsupportedPhaseDSLBlocksPublish(t *testing.T) {
	repo, db := openScriptIntegrationRepository(t)
	ctx := context.Background()
	ids := id.ULIDGenerator{}
	ownerID := ids.New()
	seedScriptUser(t, db, ownerID, "DSLOwner")
	ruleID := seedMurderRuleTemplate(t, db, ownerID)
	t.Cleanup(func() {
		cleanupScriptTestData(t, db, ownerID, "", ruleID)
	})
	service := NewService(repo)
	created, err := service.Create(ctx, ownerID, CreateScriptInput{
		Title:           "Unsupported DSL",
		GameType:        "script_murder",
		RightsType:      "original",
		Language:        "zh-CN",
		TargetPlayers:   TargetPlayers{Min: 1, Max: 1},
		DurationMinutes: 30,
		Difficulty:      "simple",
		ContentRating:   "all",
		RuleTemplateID:  &ruleID,
	}, ids.New())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	entry := json.RawMessage(`{"all":[{"fact":"approved"}]}`)
	result := mustCreateObject(
		t,
		service,
		ownerID,
		created.Script.ID,
		created.Version.ID,
		"stage",
		created.Version.RowVersion,
		StageInput{
			StageKey:        "read",
			PhaseType:       "read",
			SequenceNo:      1,
			Title:           "Read",
			EntryConditions: &entry,
			AllowedActions:  []string{"reading_complete"},
			Visibility:      "room",
		},
	)
	report, err := service.Validate(
		ctx,
		ownerID,
		created.Script.ID,
		created.Version.ID,
		ValidateInput{ExpectedRowVersion: result.VersionRowVersion},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.ErrorCode == "UNSUPPORTED_PHASE_CONDITION_DSL" &&
			issue.State == ValidationStateUnsupported {
			found = true
		}
	}
	if !found {
		t.Fatalf("unsupported DSL issue missing: %#v", report.Issues)
	}
	_, err = service.Publish(
		ctx,
		ownerID,
		created.Script.ID,
		created.Version.ID,
		PublishInput{
			Channel:            "private",
			ExpectedRowVersion: report.VersionRowVersion,
			ContentRatingAck:   true,
		},
		ids.New(),
	)
	if !hasScriptCode(err, "PUBLISH_BLOCKED") {
		t.Fatalf("Publish(unsupported DSL) error = %v", err)
	}
}

func TestScriptMetadataListPermissionsAndSoftDelete(t *testing.T) {
	repo, db := openScriptIntegrationRepository(t)
	ctx := context.Background()
	ids := id.ULIDGenerator{}
	ownerID := ids.New()
	otherID := ids.New()
	seedScriptUser(t, db, ownerID, "MetadataOwner")
	seedScriptUser(t, db, otherID, "MetadataOther")
	t.Cleanup(func() {
		cleanupScriptTestData(t, db, ownerID, otherID, "")
	})
	service := NewService(repo)
	created, err := service.Create(ctx, ownerID, CreateScriptInput{
		Title:           "Metadata Test",
		GameType:        "script_murder",
		RightsType:      "original",
		Language:        "zh-CN",
		TargetPlayers:   TargetPlayers{Min: 1, Max: 1},
		DurationMinutes: 30,
		Difficulty:      "simple",
		ContentRating:   "all",
	}, ids.New())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	page, err := service.ListPublic(ctx, "", "script_murder", 0, 0, 0, "", "", 20)
	if err != nil {
		t.Fatalf("ListPublic() error = %v", err)
	}
	for _, item := range page.Items {
		if item.ScriptID == created.Script.ID {
			t.Fatal("unpublished script appeared in public list")
		}
	}

	updated, err := service.UpdateMetadata(
		ctx,
		ownerID,
		created.Script.ID,
		UpdateScriptInput{
			RowVersion: created.Script.RowVersion,
			Patch: UpdateScriptPatch{
				Summary:       stringPointer("Updated summary"),
				Difficulty:    stringPointer("standard"),
				MinPlayers:    intPointer(1),
				MaxPlayers:    intPointer(2),
				ContentRating: stringPointer("12+"),
			},
		},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}
	if updated.Script.RowVersion != created.Script.RowVersion+1 ||
		updated.Script.Summary == nil ||
		*updated.Script.Summary != "Updated summary" {
		t.Fatalf("updated script = %#v", updated.Script)
	}

	_, err = service.UpdateMetadata(
		ctx,
		otherID,
		created.Script.ID,
		UpdateScriptInput{
			RowVersion: updated.Script.RowVersion,
			Patch: UpdateScriptPatch{
				Title: stringPointer("Unauthorized"),
			},
		},
		ids.New(),
	)
	if !hasScriptCode(err, "FORBIDDEN_EDIT") {
		t.Fatalf("UpdateMetadata(non-owner) error = %v", err)
	}

	deleted, err := service.SoftDelete(
		ctx,
		ownerID,
		created.Script.ID,
		DeleteScriptInput{RowVersion: updated.Script.RowVersion, Reason: "test"},
		ids.New(),
	)
	if err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	if deleted.Status != "soft_deleted" {
		t.Fatalf("SoftDelete() = %#v", deleted)
	}
	_, err = service.EditorContent(ctx, ownerID, created.Script.ID, created.Version.ID)
	if !hasScriptCode(err, "SCRIPT_NOT_FOUND") {
		t.Fatalf("EditorContent(deleted) error = %v", err)
	}
}

func mustCreateObject(
	t *testing.T,
	service *Service,
	ownerID string,
	scriptID string,
	versionID string,
	objectType string,
	rowVersion uint32,
	input any,
) ObjectMutationResult {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal %s: %v", objectType, err)
	}
	result, err := service.CreateObject(
		context.Background(),
		ownerID,
		scriptID,
		versionID,
		objectType,
		ObjectMutationInput{ExpectedRowVersion: rowVersion, Object: raw},
		id.ULIDGenerator{}.New(),
	)
	if err != nil {
		t.Fatalf("CreateObject(%s) error = %v", objectType, err)
	}
	return result
}

func openScriptIntegrationRepository(t *testing.T) (*storage.MySQLRepository, *database.DB) {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set")
	}
	db, err := database.Open(context.Background(), config.MySQLConfig{
		DSN:             dsn,
		MaxOpenConns:    8,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		PingTimeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return storage.NewMySQLRepository(db), db
}

func seedScriptUser(t *testing.T, db *database.DB, userID string, nickname string) {
	t.Helper()
	publicNo := "MV" + userID[len(userID)-10:]
	if _, err := db.SQL().Exec(`
INSERT INTO users (
    id, public_no, status, risk_level, terms_version, privacy_version
) VALUES (?, ?, 'active', 0, '2026-11', '2026-11')
`, userID, publicNo); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.SQL().Exec(`
INSERT INTO user_profiles (
    id, user_id, nickname, avatar_uri, bio, locale, timezone,
    preferences_json, privacy_json, moderation_status, row_version
) VALUES (?, ?, ?, NULL, NULL, 'zh-CN', 'Asia/Shanghai', ?, ?, 'approved', 1)
`, id.ULIDGenerator{}.New(), userID, nickname,
		[]byte(`{"schema_version":"1.0.0"}`),
		[]byte(`{"history_visibility":"private","online_visibility":"friends","profile_visibility":"public","friend_request_policy":"friends_of_friends"}`),
	); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
}

func seedMurderRuleTemplate(t *testing.T, db *database.DB, ownerID string) string {
	t.Helper()
	ruleID := id.ULIDGenerator{}.New()
	policy := map[string]any{
		"schema_version": "1.0.0",
		"test_nonce":     ruleID,
		"murder": map[string]any{
			"schema_version":         "1.0.0",
			"default_search_quota":   1,
			"allow_abstain":          true,
			"allow_revote":           false,
			"allow_self_vote":        false,
			"vote_visibility":        "count_only",
			"tie_policy":             "no_elimination",
			"culprit_character_keys": []string{"killer"},
			"voting_character_keys":  []string{"killer", "civilian"},
			"role_factions": map[string]string{
				"killer":   "killer",
				"civilian": "civilian",
			},
			"role_teams": map[string]string{},
		},
	}
	raw, _ := json.Marshal(policy)
	hash := sha256.Sum256(raw)
	if _, err := db.SQL().Exec(`
INSERT INTO rule_templates (
    id, template_key, version_no, game_type, name, status, schema_version,
    engine_version, config_json, owner_user_id, content_hash, published_at
) VALUES (?, ?, 1, 'script_murder', 'Task 7 Rule', 'active', '1.0.0',
          'murder-engine-1.0.0', ?, ?, ?, UTC_TIMESTAMP(3))
`, ruleID, "task7-"+strings.ToLower(ruleID), raw, ownerID, hash[:]); err != nil {
		t.Fatalf("seed rule template: %v", err)
	}
	return ruleID
}

func cleanupScriptTestData(
	t *testing.T,
	db *database.DB,
	ownerID string,
	otherID string,
	ruleID string,
) {
	t.Helper()
	execDelete(t, db, `
DELETE spc FROM script_search_point_clues spc
JOIN script_versions sv ON sv.id = spc.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE sp FROM script_search_points sp
JOIN script_versions sv ON sv.id = sp.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE sc FROM script_clues sc
JOIN script_versions sv ON sv.id = sc.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE sc FROM script_characters sc
JOIN script_versions sv ON sv.id = sc.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE ph FROM script_phases ph
JOIN script_versions sv ON sv.id = ph.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE se FROM script_endings se
JOIN script_versions sv ON sv.id = se.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE sa FROM script_assets sa
JOIN script_versions sv ON sv.id = sa.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE ch FROM script_chapters ch
JOIN script_versions sv ON sv.id = ch.script_version_id
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE p FROM script_publications p
JOIN scripts s ON s.id = p.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
UPDATE script_versions sv
JOIN scripts s ON s.id = sv.script_id
SET sv.base_version_id = NULL
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, `
DELETE sv FROM script_versions sv
JOIN scripts s ON s.id = sv.script_id
WHERE s.owner_user_id = ?`, ownerID)
	execDelete(t, db, "DELETE FROM scripts WHERE owner_user_id = ?", ownerID)
	execDelete(t, db, "DELETE FROM idempotency_records WHERE actor_scope = ?", "user:"+ownerID)
	execDelete(t, db, "DELETE FROM rule_templates WHERE id = ?", ruleID)
	execDelete(t, db, "DELETE FROM user_profiles WHERE user_id = ?", ownerID)
	execDelete(t, db, "DELETE FROM user_profiles WHERE user_id = ?", otherID)
	execDelete(t, db, "DELETE FROM users WHERE id = ?", ownerID)
	execDelete(t, db, "DELETE FROM users WHERE id = ?", otherID)
}

func execDelete(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.SQL().Exec(query, args...); err != nil {
		t.Fatalf("cleanup query failed: %v", err)
	}
}

func hasScriptCode(err error, code string) bool {
	var appErr apperror.Error
	return errors.As(err, &appErr) && appErr.Code == code
}

func intPointer(value int) *int {
	return &value
}
