package domain

import (
	"encoding/json"
	"testing"
	"time"
)

type fixedRandomizer struct {
	permutation []int
}

func (r fixedRandomizer) Permutation(size int) ([]int, error) {
	return append([]int(nil), r.permutation[:size]...), nil
}

func TestStatusTransitions(t *testing.T) {
	if !StatusStarting.CanTransitionTo(StatusRunning) {
		t.Fatal("starting must transition to running")
	}
	if !StatusRunning.CanTransitionTo(StatusPaused) {
		t.Fatal("running must transition to paused")
	}
	if !StatusPaused.CanTransitionTo(StatusRunning) {
		t.Fatal("paused must transition to running")
	}
	if StatusCompleted.CanTransitionTo(StatusRunning) {
		t.Fatal("completed must be terminal")
	}
	if StatusRunning.CanTransitionTo(StatusCompleted) {
		t.Fatal("running cannot skip settling")
	}
}

func TestPhaseSequenceCannotSkip(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()

	if err := engine.Validate(state, config, Actor{
		PlayerID: "p1",
		RoleID:   "r1",
	}, Action{
		Type: ActionVote,
		Payload: map[string]any{
			"target_player_id": "p2",
		},
	}, time.Now().UTC()); err == nil {
		t.Fatal("vote was accepted during read phase")
	}
}

func TestRoleAssignmentAndPrivateActorValidation(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	players := []Player{
		{ID: "p1", UserID: "u1", RuntimeStatus: "active"},
		{ID: "p2", UserID: "u2", RuntimeStatus: "active"},
	}
	roles, err := engine.AssignRoles(players, config, fixedRandomizer{permutation: []int{0, 1}})
	if err != nil {
		t.Fatalf("AssignRoles() error = %v", err)
	}
	if len(roles) != 2 || roles[0].PlayerID == roles[1].PlayerID {
		t.Fatalf("roles = %#v", roles)
	}
	if roles[0].FactionKey == "" || roles[1].CharacterID == "" {
		t.Fatalf("role assignment missing data: %#v", roles)
	}
}

func TestReadingCompletionAdvancesPhase(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	now := time.Now().UTC()
	state.PhaseStartedAt = now

	first, err := engine.Apply(state, config, Actor{PlayerID: "p1", RoleID: "r1"}, Action{
		Type: ActionReadingComplete,
	}, now)
	if err != nil {
		t.Fatalf("first reading complete error = %v", err)
	}
	if first.State.PhaseKey != "read" {
		t.Fatalf("phase advanced to %q too early", first.State.PhaseKey)
	}

	second, err := engine.Apply(first.State, config, Actor{PlayerID: "p2", RoleID: "r2"}, Action{
		Type: ActionReadingComplete,
	}, now)
	if err != nil {
		t.Fatalf("second reading complete error = %v", err)
	}
	if second.State.PhaseKey != "opening" {
		t.Fatalf("phase = %q, want opening", second.State.PhaseKey)
	}
}

func TestVoteRevoteAndSettlement(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	now := time.Now().UTC()
	state.PhaseIndex = 3
	state.PhaseKey = "vote"
	state.PhaseRoundKey = "vote"
	state.PhaseStartedAt = now

	first, err := engine.Apply(state, config, Actor{PlayerID: "p1", RoleID: "r1"}, Action{
		Type: ActionVote,
		Payload: map[string]any{
			"target_player_id": "p1",
		},
	}, now)
	if err != nil {
		t.Fatalf("first vote error = %v", err)
	}
	second, err := engine.Apply(first.State, config, Actor{PlayerID: "p1", RoleID: "r1"}, Action{
		Type: ActionVote,
		Payload: map[string]any{
			"target_player_id": "p2",
		},
	}, now)
	if err != nil {
		t.Fatalf("revote error = %v", err)
	}
	if len(second.State.Votes) != 1 {
		t.Fatalf("votes = %d, want one upserted vote", len(second.State.Votes))
	}

	final, err := engine.Apply(second.State, config, Actor{PlayerID: "p2", RoleID: "r2"}, Action{
		Type: ActionVote,
		Payload: map[string]any{
			"target_player_id": "p1",
		},
	}, now)
	if err != nil {
		t.Fatalf("second vote error = %v", err)
	}
	if final.State.Status != StatusSettling {
		t.Fatalf("status = %q, want settling", final.State.Status)
	}
}

func TestSearchPersistsExclusiveClaimInState(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	now := time.Now().UTC()
	state.PhaseIndex = 2
	state.PhaseKey = "search"
	state.PhaseRoundKey = "search"
	state.PhaseStartedAt = now

	first, err := engine.Apply(state, config, Actor{PlayerID: "p1", RoleID: "r1"}, Action{
		Type: ActionSearch,
		Payload: map[string]any{
			"search_point_id": "point1",
		},
	}, now)
	if err != nil {
		t.Fatalf("first Search() error = %v", err)
	}
	if len(first.State.FoundClues) != 1 {
		t.Fatalf("FoundClues = %d, want 1", len(first.State.FoundClues))
	}
	_, err = engine.Apply(first.State, config, Actor{PlayerID: "p2", RoleID: "r2"}, Action{
		Type: ActionSearch,
		Payload: map[string]any{
			"search_point_id": "point1",
		},
	}, now)
	if err == nil {
		t.Fatal("second Search() claimed an exclusive clue")
	}
}

func TestCheckEnd(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	state.PhaseIndex = 3
	state.PhaseKey = "vote"
	state.PhaseRoundKey = "vote"
	state.Votes = []Vote{
		{RoundKey: "vote", VoterPlayerID: "p1", TargetPlayerID: "p1"},
		{RoundKey: "vote", VoterPlayerID: "p2", TargetPlayerID: "p1"},
	}
	end, completed, err := engine.CheckEnd(state, config)
	if err != nil {
		t.Fatalf("CheckEnd() error = %v", err)
	}
	if !completed || end.ResultCode != "murder_solved" || end.WinnerTeam != "innocents" {
		t.Fatalf("CheckEnd() = %#v", end)
	}
}

func TestStateHashIsStable(t *testing.T) {
	state := testRunningState()
	first := StateHash(state)
	second := StateHash(state)
	if !EqualHash(first, second) {
		t.Fatal("StateHash is not stable")
	}
	state.StateVersion++
	if EqualHash(first, StateHash(state)) {
		t.Fatal("StateHash did not change after state mutation")
	}
}

func TestInvalidPolicyIsRejected(t *testing.T) {
	config := testConfig()
	config.Policy.TiePolicy = "unknown_policy"
	if _, err := NewScriptMurderEngine(config); err == nil {
		t.Fatal("NewScriptMurderEngine() accepted an unknown tie policy")
	}
}

func TestTickFallbackAdvancesExpiredPhase(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	now := time.Now().UTC()
	state.PhaseIndex = 1
	state.PhaseKey = "opening"
	state.PhaseRoundKey = "opening"
	state.PhaseStartedAt = now.Add(-2 * time.Minute)
	deadline := now.Add(-time.Minute)
	state.PhaseDeadlineAt = &deadline

	result, err := engine.Tick(state, config, now)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if result.State.PhaseKey != "search" || len(result.Events) == 0 {
		t.Fatalf("Tick() = %#v", result)
	}
}

func TestTickVoteTimeoutSettles(t *testing.T) {
	engine := testEngine(t)
	config := testConfig()
	state := testRunningState()
	now := time.Now().UTC()
	state.PhaseIndex = 3
	state.PhaseKey = "vote"
	state.PhaseRoundKey = "vote"
	deadline := now.Add(-time.Minute)
	state.PhaseDeadlineAt = &deadline
	state.Votes = []Vote{
		{RoundKey: "vote", VoterPlayerID: "p1", TargetPlayerID: "p1"},
	}

	result, err := engine.Tick(state, config, now)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}
	if result.State.Status != StatusSettling || result.ResultCode == "" {
		t.Fatalf("Tick() = %#v", result)
	}
}

func testEngine(t *testing.T) *ScriptMurderEngine {
	t.Helper()
	engine, err := NewScriptMurderEngine(testConfig())
	if err != nil {
		t.Fatalf("NewScriptMurderEngine() error = %v", err)
	}
	return engine
}

func testConfig() ScriptConfig {
	timeout := 60
	return ScriptConfig{
		SchemaVersion: "1.0.0",
		EngineKey:     EngineKey,
		EngineVersion: EngineVersion,
		Policy: MurderPolicy{
			SchemaVersion:      "1.0.0",
			DefaultSearchQuota: 1,
			AllowAbstain:       true,
			AllowRevote:        true,
			AllowSelfVote:      true,
			VoteVisibility:     "count_only",
			TiePolicy:          "no_elimination",
			CulpritCharacterKeys: []string{
				"char_culprit",
			},
			VotingCharacterKeys: []string{
				"char_culprit",
				"char_victim",
			},
			RoleFactions: map[string]string{
				"char_culprit": "culprit",
				"char_victim":  "civilian",
			},
		},
		Phases: []Phase{
			{Key: "read", Kind: PhaseRead, Sequence: 1, AllowedActions: []ActionType{ActionReadingComplete}, TimeoutSeconds: &timeout},
			{Key: "opening", Kind: PhaseOpening, Sequence: 2, AllowedActions: []ActionType{ActionAdvancePhase}, TimeoutSeconds: &timeout},
			{Key: "search", Kind: PhaseSearch, Sequence: 3, AllowedActions: []ActionType{ActionSearch}, TimeoutSeconds: &timeout},
			{Key: "vote", Kind: PhaseVote, Sequence: 4, AllowedActions: []ActionType{ActionVote}, TimeoutSeconds: &timeout},
		},
		Characters: []Character{
			{
				ID:            "c1",
				Key:           "char_culprit",
				DisplayName:   "Culprit",
				PublicProfile: "The culprit",
				PublicGoals:   json.RawMessage(`[]`),
				Attributes:    json.RawMessage(`{}`),
			},
			{
				ID:            "c2",
				Key:           "char_victim",
				DisplayName:   "Victim",
				PublicProfile: "The victim",
				PublicGoals:   json.RawMessage(`[]`),
				Attributes:    json.RawMessage(`{}`),
			},
		},
		Clues: []Clue{
			{
				ID:              "clue1",
				Key:             "clue_1",
				Title:           "Clue",
				ClueType:        "physical",
				PublicText:      "A clue",
				VisibilityScope: "public",
			},
		},
		SearchPoints: []SearchPoint{
			{
				ID:             "point1",
				Key:            "point_1",
				PhaseKey:       "search",
				Title:          "Point",
				ConflictPolicy: "first_wins",
				MaxSearches:    1,
			},
		},
		PointClues: []SearchPointClue{
			{
				SearchPointID: "point1",
				ClueID:        "clue1",
				Weight:        1,
				IsDefault:     true,
			},
		},
	}
}

func testRunningState() RuntimeState {
	return RuntimeState{
		GameID:       "g1",
		GameType:     "script_murder",
		Status:       StatusRunning,
		StateVersion: 1,
		EventSeq:     1,
		PhaseIndex:   0,
		PhaseKey:     "read",
		Players: []Player{
			{ID: "p1", UserID: "u1", RuntimeStatus: "active"},
			{ID: "p2", UserID: "u2", RuntimeStatus: "active"},
		},
		Roles: []Role{
			{ID: "r1", PlayerID: "p1", CharacterID: "c1", CharacterKey: "char_culprit", FactionKey: "culprit"},
			{ID: "r2", PlayerID: "p2", CharacterID: "c2", CharacterKey: "char_victim", FactionKey: "civilian"},
		},
		SearchPointUses: make(map[string]int),
	}
}
