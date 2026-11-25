package compile

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mysteryverse/mysteryverse/backend/internal/game/domain"
	"github.com/mysteryverse/mysteryverse/backend/internal/script"
)

type Result struct {
	Config      domain.ScriptConfig
	PackageHash []byte
}

type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(
	loaded script.PlayableVersion,
) (Result, error) {
	if loaded.RuleTemplate == nil {
		return Result{}, errors.New("a rule template with murder policy is required")
	}
	policy, err := parsePolicy(loaded.RuleTemplate.Config)
	if err != nil {
		return Result{}, err
	}

	phases, phaseKeys, err := compilePhases(loaded.Content.Stages)
	if err != nil {
		return Result{}, err
	}
	if !containsPhaseKind(phases, domain.PhaseVote) {
		return Result{}, errors.New("script must define a vote phase")
	}

	characters := make([]domain.Character, 0, len(loaded.Content.Characters))
	for _, row := range loaded.Content.Characters {
		characters = append(characters, domain.Character{
			ID:               row.ID,
			Key:              row.CharacterKey,
			DisplayName:      row.DisplayName,
			RoleClass:        row.RoleClass,
			PublicProfile:    row.PublicProfile,
			PublicGoals:      mustJSON(row.PublicGoals),
			Attributes:       rawJSON(row.Attributes),
			PrivatePayloadID: stringValue(row.PrivatePayloadID),
		})
	}
	if len(characters) == 0 {
		return Result{}, errors.New("script must define characters")
	}
	for _, character := range characters {
		if policy.RoleFactions[character.Key] == "" {
			return Result{}, fmt.Errorf("role faction is missing for character %q", character.Key)
		}
	}

	clues := make([]domain.Clue, 0, len(loaded.Content.Clues))
	for _, row := range loaded.Content.Clues {
		clue := domain.Clue{
			ID:               row.ID,
			Key:              row.ClueKey,
			Title:            row.Title,
			ClueType:         row.ClueType,
			IsRepeatable:     row.IsRepeatable,
			VisibilityScope:  row.Visibility,
			Sensitivity:      sensitivityNumber(row.Sensitivity),
			PrivatePayloadID: stringValue(row.PrivatePayloadID),
		}
		if row.PublicText != nil {
			clue.PublicText = *row.PublicText
		}
		if row.MaxDiscoveries != nil {
			value := *row.MaxDiscoveries
			clue.MaxDiscoveries = &value
		}
		clues = append(clues, clue)
	}

	points := make([]domain.SearchPoint, 0, len(loaded.Content.SearchPoints))
	for _, row := range loaded.Content.SearchPoints {
		phaseKey := phaseKeys[row.StageID]
		if phaseKey == "" {
			return Result{}, fmt.Errorf("search point %s references an unknown phase", row.ID)
		}
		points = append(points, domain.SearchPoint{
			ID:                row.ID,
			Key:               row.PointKey,
			PhaseKey:          phaseKey,
			Title:             row.Title,
			PublicDescription: row.PublicDescription,
			MaxSearches:       row.MaxSearches,
			ConflictPolicy:    row.ConflictPolicy,
		})
	}

	pointClues := make([]domain.SearchPointClue, 0, len(loaded.Content.PointClues))
	for _, row := range loaded.Content.PointClues {
		pointClues = append(pointClues, domain.SearchPointClue{
			SearchPointID: row.SearchPointID,
			ClueID:        row.ClueID,
			Weight:        row.Weight,
			IsDefault:     row.IsDefault,
		})
	}

	config := domain.ScriptConfig{
		ScriptVersionID: loaded.Version.ID,
		Title:           loaded.Script.Title,
		EngineKey:       domain.EngineKey,
		EngineVersion:   domain.EngineVersion,
		SchemaVersion:   loaded.Version.SchemaVersion,
		Policy:          policy,
		Phases:          phases,
		Characters:      characters,
		Clues:           clues,
		SearchPoints:    points,
		PointClues:      pointClues,
	}
	if _, err := domain.NewScriptMurderEngine(config); err != nil {
		return Result{}, fmt.Errorf("validate compiled script: %w", err)
	}
	raw, _ := json.Marshal(config)
	digest := sha256.Sum256(raw)
	return Result{Config: config, PackageHash: digest[:]}, nil
}

func parsePolicy(raw json.RawMessage) (domain.MurderPolicy, error) {
	var envelope struct {
		SchemaVersion string              `json:"schema_version"`
		Murder        domain.MurderPolicy `json:"murder"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.MurderPolicy{}, fmt.Errorf("parse rule template config: %w", err)
	}
	policy := envelope.Murder
	if policy.SchemaVersion == "" {
		policy.SchemaVersion = envelope.SchemaVersion
	}
	if err := policy.Validate(); err != nil {
		return domain.MurderPolicy{}, fmt.Errorf("validate murder policy: %w", err)
	}
	return policy, nil
}

func compilePhases(rows []script.Stage) ([]domain.Phase, map[string]string, error) {
	phases := make([]domain.Phase, 0, len(rows))
	phaseKeys := make(map[string]string, len(rows))
	for _, row := range rows {
		if hasNonEmptyConditions(row.EntryConditions) ||
			hasNonEmptyConditions(row.ExitConditions) {
			return nil, nil, fmt.Errorf(
				"phase %q uses unsupported entry/exit condition DSL",
				row.Key,
			)
		}
		actions := make([]domain.ActionType, 0, len(row.AllowedActions))
		for _, rawAction := range row.AllowedActions {
			action, err := parseAction(rawAction)
			if err != nil {
				return nil, nil, fmt.Errorf("stage %q: %w", row.Key, err)
			}
			actions = append(actions, action)
		}
		if len(actions) == 0 {
			return nil, nil, fmt.Errorf("stage %q has no allowed actions", row.Key)
		}
		kind := domain.PhaseKind(row.Type)
		if err := kind.Validate(); err != nil {
			return nil, nil, fmt.Errorf("stage %q: %w", row.Key, err)
		}
		phases = append(phases, domain.Phase{
			ID:             row.ID,
			Key:            row.Key,
			Kind:           kind,
			Sequence:       row.SequenceNo,
			Title:          row.Title,
			AllowedActions: actions,
			TimeoutSeconds: row.TimeoutSeconds,
			FallbackAction: row.FallbackAction,
			IsTerminal:     row.IsTerminal,
		})
		phaseKeys[row.ID] = row.Key
	}
	_, err := domain.NewStateMachine(phases)
	if err != nil {
		return nil, nil, fmt.Errorf("validate phase sequence: %w", err)
	}
	return phases, phaseKeys, nil
}

func parseAction(value string) (domain.ActionType, error) {
	action := domain.ActionType(value)
	switch action {
	case domain.ActionReadingComplete, domain.ActionSearch, domain.ActionVote,
		domain.ActionAdvancePhase:
		return action, nil
	default:
		return "", fmt.Errorf("unsupported action %q", value)
	}
}

func containsPhaseKind(phases []domain.Phase, kind domain.PhaseKind) bool {
	for _, phase := range phases {
		if phase.Kind == kind {
			return true
		}
	}
	return false
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func rawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("{}")
	}
	return append(json.RawMessage(nil), value...)
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func sensitivityNumber(value string) int {
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

func hasNonEmptyConditions(value *json.RawMessage) bool {
	if value == nil || len(*value) == 0 {
		return false
	}
	var object map[string]any
	if err := json.Unmarshal(*value, &object); err != nil {
		return true
	}
	return len(object) > 0
}
