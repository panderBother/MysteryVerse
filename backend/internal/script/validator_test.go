package script

import (
	"encoding/json"
	"testing"
)

func TestValidatorDetectsReferencesCountsAndUnsupportedDSL(t *testing.T) {
	validator := NewValidator()
	content := Content{
		Script: Script{
			ID:              "01J9Z3K4M7Q2R8T5V6W1X10000",
			Title:           "Validator Test",
			GameType:        "script_murder",
			Language:        "zh-CN",
			ContentRating:   "16+",
			MinPlayers:      2,
			MaxPlayers:      4,
			DurationMinutes: 90,
		},
		Version: Version{
			ID:            "01J9Z3K4M7Q2R8T5V6W1X20000",
			SchemaVersion: "1.0.0",
			RowVersion:    4,
		},
		Chapters: []Chapter{{
			ID:         "01J9Z3K4M7Q2R8T5V6W1X30000",
			ChapterKey: "chapter-1",
			Title:      "Chapter 1",
			SequenceNo: 1,
			SourceRefs: json.RawMessage("[]"),
		}},
		Stages: []Stage{
			{
				ID:         "01J9Z3K4M7Q2R8T5V6W1X40000",
				ChapterID:  stringPointer("01J9Z3K4M7Q2R8T5V6W1X9999"),
				Key:        "read",
				Type:       "read",
				SequenceNo: 1,
				Title:      "Read",
				AllowedActions: []string{
					"reading_complete",
				},
				Visibility: "room",
			},
			{
				ID:             "01J9Z3K4M7Q2R8T5V6W1X50000",
				Key:            "vote",
				Type:           "vote",
				SequenceNo:     2,
				Title:          "Vote",
				AllowedActions: []string{"vote"},
				Visibility:     "room",
			},
		},
		Characters: []Character{{
			ID:            "01J9Z3K4M7Q2R8T5V6W1X60000",
			CharacterKey:  "crime",
			DisplayName:   "Only Character",
			RoleClass:     "player_role",
			PublicProfile: "profile",
			PublicGoals:   []string{"goal"},
			Visibility:    "role",
			Sensitivity:   "D2",
			Attributes:    json.RawMessage("{}"),
			SourceRefs:    json.RawMessage("[]"),
		}},
		Clues: []Clue{{
			ID:          "01J9Z3K4M7Q2R8T5V6W1X70000",
			ClueKey:     "private-clue",
			Title:       "Private Clue",
			PublicText:  stringPointer("hidden"),
			ClueType:    "physical",
			Visibility:  "role",
			Sensitivity: "D2",
			SourceRefs:  json.RawMessage("[]"),
		}},
		Endings: []Ending{{
			ID:          "01J9Z3K4M7Q2R8T5V6W1X80000",
			EndingKey:   "ending-1",
			Title:       "Ending",
			Description: "description",
			Conditions:  json.RawMessage(`{"all":[{"fact":"a"}]}`),
			Visibility:  "room",
		}},
	}
	report := ValidationReport{Issues: make([]ValidationIssue, 0)}
	validator.validateMetadata(content, &report)
	validator.validateChapters(content, &report)
	validator.validateStages(content, &report)
	validator.validateCharacters(content, &report)
	validator.validateClues(content, &report)
	validator.validateEndings(content, &report)
	finalizeReport(&report)

	requiredCodes := map[string]bool{
		"STAGE_CHAPTER_REFERENCE_INVALID":  false,
		"CHARACTER_COUNT_OUT_OF_RANGE":     false,
		"CLUE_UNREACHABLE":                 false,
		"UNSUPPORTED_ENDING_CONDITION_DSL": false,
	}
	for _, issue := range report.Issues {
		if _, exists := requiredCodes[issue.ErrorCode]; exists {
			requiredCodes[issue.ErrorCode] = true
		}
	}
	for code, found := range requiredCodes {
		if !found {
			t.Fatalf("validator did not report %s in %#v", code, report.Issues)
		}
	}
	if report.Status != "blocked" || report.BlockingCount < 4 {
		t.Fatalf("report = %#v", report)
	}

	unsupported := 0
	for _, issue := range report.Issues {
		if issue.State == ValidationStateUnsupported {
			unsupported++
		}
	}
	if unsupported != 1 {
		t.Fatalf("unsupported issue count = %d", unsupported)
	}
}

func TestValidatorRequiresSearchPointReachability(t *testing.T) {
	validator := NewValidator()
	content := Content{
		Stages: []Stage{{
			ID:         "01J9Z3K4M7Q2R8T5V6W1X40000",
			Key:        "search",
			Type:       "search",
			SequenceNo: 1,
			Title:      "Search",
			AllowedActions: []string{
				"search",
			},
			Visibility: "room",
		}},
		Clues: []Clue{{
			ID:         "01J9Z3K4M7Q2R8T5V6W1X70000",
			ClueKey:    "unreachable",
			Title:      "Unreachable",
			ClueType:   "physical",
			Visibility: "role",
		}},
	}
	report := ValidationReport{Issues: make([]ValidationIssue, 0)}
	validator.validateClues(content, &report)
	validator.validateSearchPoints(content, &report)
	finalizeReport(&report)
	hasUnreachable := false
	hasSearchPointsRequired := false
	for _, issue := range report.Issues {
		hasUnreachable = hasUnreachable || issue.ErrorCode == "CLUE_UNREACHABLE"
		hasSearchPointsRequired = hasSearchPointsRequired ||
			issue.ErrorCode == "SEARCH_POINTS_REQUIRED"
	}
	if !hasUnreachable || !hasSearchPointsRequired {
		t.Fatalf("issues = %#v", report.Issues)
	}
}
