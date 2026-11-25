package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	EngineKey     = "murder"
	EngineVersion = "murder-engine-1.0.0"
)

type Randomizer interface {
	Permutation(size int) ([]int, error)
}

type Engine interface {
	Key() string
	Version() string
	AssignRoles(players []Player, config ScriptConfig, randomizer Randomizer) ([]Role, error)
	Validate(state RuntimeState, config ScriptConfig, actor Actor, action Action, now time.Time) error
	Apply(
		state RuntimeState,
		config ScriptConfig,
		actor Actor,
		action Action,
		now time.Time,
	) (DecisionResult, error)
	Tick(state RuntimeState, config ScriptConfig, now time.Time) (DecisionResult, error)
	CheckEnd(state RuntimeState, config ScriptConfig) (EndResult, bool, error)
}

type ScriptMurderEngine struct {
	machine *StateMachine
}

func NewScriptMurderEngine(config ScriptConfig) (*ScriptMurderEngine, error) {
	if config.SchemaVersion == "" {
		return nil, errors.New("script schema_version is required")
	}
	if err := config.Policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid murder policy: %w", err)
	}
	if len(config.Characters) == 0 {
		return nil, errors.New("script must define characters")
	}
	machine, err := NewStateMachine(config.Phases)
	if err != nil {
		return nil, err
	}
	return &ScriptMurderEngine{machine: machine}, nil
}

func (*ScriptMurderEngine) Key() string {
	return EngineKey
}

func (*ScriptMurderEngine) Version() string {
	return EngineVersion
}

func (e *ScriptMurderEngine) AssignRoles(
	players []Player,
	config ScriptConfig,
	randomizer Randomizer,
) ([]Role, error) {
	if len(players) != len(config.Characters) {
		return nil, fmt.Errorf(
			"player count %d does not match character count %d",
			len(players),
			len(config.Characters),
		)
	}
	permutation, err := randomizer.Permutation(len(config.Characters))
	if err != nil {
		return nil, fmt.Errorf("generate role permutation: %w", err)
	}
	roles := make([]Role, 0, len(players))
	for index, player := range players {
		characterIndex := permutation[index]
		if characterIndex < 0 || characterIndex >= len(config.Characters) {
			return nil, errors.New("randomizer returned an invalid permutation")
		}
		character := config.Characters[characterIndex]
		faction := config.Policy.RoleFactions[character.Key]
		if faction == "" {
			return nil, fmt.Errorf("role faction is missing for character %q", character.Key)
		}
		roles = append(roles, Role{
			PlayerID:         player.ID,
			CharacterID:      character.ID,
			CharacterKey:     character.Key,
			DisplayName:      character.DisplayName,
			FactionKey:       faction,
			TeamKey:          config.Policy.RoleTeams[character.Key],
			PublicProfile:    character.PublicProfile,
			RevealState:      "hidden",
			PrivatePayloadID: character.PrivatePayloadID,
		})
	}
	return roles, nil
}

func (e *ScriptMurderEngine) Validate(
	state RuntimeState,
	config ScriptConfig,
	actor Actor,
	action Action,
	now time.Time,
) error {
	if state.Status != StatusRunning {
		return fmt.Errorf("game is not running")
	}
	player, role, found := findActor(state, actor)
	if !found || player.RuntimeStatus != "active" {
		return errors.New("actor is not an active game player")
	}
	phase, err := e.machine.Current(state)
	if err != nil {
		return err
	}
	if !containsAction(phase.AllowedActions, action.Type) {
		return fmt.Errorf("action %q is not allowed in phase %q", action.Type, phase.Key)
	}
	switch action.Type {
	case ActionReadingComplete:
		if phase.Kind != PhaseRead {
			return errors.New("reading completion is only valid during a read phase")
		}
	case ActionSearch:
		if phase.Kind != PhaseSearch {
			return errors.New("search is only valid during a search phase")
		}
		pointID, _ := action.Payload["search_point_id"].(string)
		point, ok := findSearchPoint(config, pointID)
		if !ok || point.PhaseKey != phase.Key {
			return errors.New("search point is unavailable in the current phase")
		}
		quota := state.SearchPointUses["quota:"+player.ID+":"+phase.Key]
		if quota >= config.Policy.DefaultSearchQuota {
			return errors.New("search quota is exhausted")
		}
	case ActionVote:
		if phase.Kind != PhaseVote {
			return errors.New("vote is only valid during a vote phase")
		}
		if !containsString(config.Policy.VotingCharacterKeys, role.CharacterKey) {
			return errors.New("role does not have voting rights")
		}
		abstain, _ := action.Payload["abstain"].(bool)
		if abstain && !config.Policy.AllowAbstain {
			return errors.New("abstaining is not allowed")
		}
		targetID, _ := action.Payload["target_player_id"].(string)
		if !abstain {
			target, ok := findPlayer(state, targetID)
			if !ok || target.RuntimeStatus != "active" {
				return errors.New("vote target is not an active player")
			}
			if !config.Policy.AllowSelfVote && targetID == player.ID {
				return errors.New("self voting is not allowed")
			}
		}
		if !config.Policy.AllowRevote && hasVote(state, state.PhaseRoundKey, player.ID) {
			return errors.New("revoting is not allowed")
		}
	case ActionAdvancePhase:
		if !actor.IsHost {
			return errors.New("only the host may advance the phase")
		}
		if phase.Kind == PhaseVote || phase.Kind == PhaseSettlement {
			return errors.New("vote and settlement phases cannot be manually advanced")
		}
	}
	if state.PhaseDeadlineAt != nil && !now.Before(*state.PhaseDeadlineAt) {
		return errors.New("phase deadline has passed")
	}
	return nil
}

func (e *ScriptMurderEngine) Apply(
	state RuntimeState,
	config ScriptConfig,
	actor Actor,
	action Action,
	now time.Time,
) (DecisionResult, error) {
	if err := e.Validate(state, config, actor, action, now); err != nil {
		return DecisionResult{}, err
	}
	state = cloneState(state)
	phase, _ := e.machine.Current(state)

	switch action.Type {
	case ActionReadingComplete:
		setReadingCompleted(&state, actor.PlayerID)
		allComplete := allPlayersReadingCompleted(state)
		event := actionEvent("game.reading_completed", actor, map[string]any{
			"schema_version": "1.0.0",
			"player_id":      actor.PlayerID,
		})
		if allComplete {
			advanced, events, _, err := e.machine.Advance(state, now)
			if err != nil {
				return DecisionResult{}, err
			}
			state = advanced
			events = append([]DomainEvent{event}, events...)
			return DecisionResult{
				State:      state,
				Events:     events,
				ResultCode: "reading_complete_advanced",
			}, nil
		}
		return DecisionResult{State: state, Events: []DomainEvent{event}, ResultCode: "reading_completed"}, nil

	case ActionSearch:
		pointID, _ := action.Payload["search_point_id"].(string)
		result, event, err := resolveSearch(state, config, actor.PlayerID, pointID, now)
		if err != nil {
			return DecisionResult{}, err
		}
		result.Events = []DomainEvent{event}
		result.ResultCode = "search_succeeded"
		if allSearchQuotasExhausted(result.State, config, phase.Key) {
			advanced, events, _, err := e.machine.Advance(result.State, now)
			if err != nil {
				return DecisionResult{}, err
			}
			result.State = advanced
			result.Events = append(result.Events, events...)
			result.ResultCode = "search_phase_completed"
		}
		return result, nil

	case ActionVote:
		abstain, _ := action.Payload["abstain"].(bool)
		targetID, _ := action.Payload["target_player_id"].(string)
		vote := Vote{
			RoundKey:       state.PhaseRoundKey,
			BallotType:     "murder_final",
			VoterPlayerID:  actor.PlayerID,
			TargetPlayerID: targetID,
			Abstain:        abstain,
			SubmittedAt:    now,
		}
		upsertVote(&state, vote)
		event := actionEvent("game.vote_submitted", actor, map[string]any{
			"schema_version":   "1.0.0",
			"round_key":        vote.RoundKey,
			"target_player_id": targetID,
			"abstain":          abstain,
		})
		if allEligibleVotesSubmitted(state, config) {
			end, complete, err := e.CheckEnd(state, config)
			if err != nil {
				return DecisionResult{}, err
			}
			if complete {
				state.Status = StatusSettling
				state.ResultCode = end.ResultCode
				state.SettlementSummary = end.Summary
				return DecisionResult{
					State:         state,
					Events:        []DomainEvent{event},
					ResultCode:    end.ResultCode,
					ResultSummary: end.Summary,
				}, nil
			}
		}
		return DecisionResult{State: state, Events: []DomainEvent{event}, ResultCode: "vote_submitted"}, nil

	case ActionAdvancePhase:
		advanced, events, exists, err := e.machine.Advance(state, now)
		if err != nil {
			return DecisionResult{}, err
		}
		if !exists {
			return DecisionResult{}, errors.New("no next phase exists")
		}
		return DecisionResult{State: advanced, Events: events, ResultCode: "phase_advanced"}, nil
	default:
		return DecisionResult{}, fmt.Errorf("unsupported action %q", action.Type)
	}
}

func (e *ScriptMurderEngine) Tick(
	state RuntimeState,
	config ScriptConfig,
	now time.Time,
) (DecisionResult, error) {
	if state.Status != StatusRunning || state.PhaseDeadlineAt == nil ||
		now.Before(*state.PhaseDeadlineAt) {
		return DecisionResult{State: state}, nil
	}
	phase, err := e.machine.Current(state)
	if err != nil {
		return DecisionResult{}, err
	}
	if phase.Kind == PhaseVote {
		end, complete, err := e.CheckEnd(state, config)
		if err != nil {
			return DecisionResult{}, err
		}
		if complete {
			state.Status = StatusSettling
			state.ResultCode = end.ResultCode
			state.SettlementSummary = end.Summary
			return DecisionResult{
				State:         state,
				Events:        []DomainEvent{systemEvent("game.vote_timeout", end.Summary)},
				ResultCode:    end.ResultCode,
				ResultSummary: end.Summary,
			}, nil
		}
	}
	advanced, events, exists, err := e.machine.Advance(state, now)
	if err != nil {
		return DecisionResult{}, err
	}
	if !exists {
		state.Status = StatusAborted
		state.ResultCode = "aborted_no_transition"
		state.SettlementSummary = map[string]any{
			"reason":    "no_next_phase",
			"phase_key": phase.Key,
		}
		return DecisionResult{
			State: state,
			Events: []DomainEvent{
				systemEvent("game.phase_timeout", map[string]any{"phase_key": phase.Key}),
				{
					EventType:       "game.aborted",
					EventCategory:   "incident",
					VisibilityScope: "room",
					Sensitivity:     0,
					Payload: map[string]any{
						"schema_version": "1.0.0",
						"reason":         "no_next_phase",
						"phase_key":      phase.Key,
					},
					StateChanged: true,
				},
			},
			ResultCode:    "aborted_no_transition",
			ResultSummary: state.SettlementSummary,
		}, nil
	}
	return DecisionResult{State: advanced, Events: events, ResultCode: "phase_timeout_advanced"}, nil
}

func (e *ScriptMurderEngine) CheckEnd(
	state RuntimeState,
	config ScriptConfig,
) (EndResult, bool, error) {
	phase, err := e.machine.Current(state)
	if err != nil {
		return EndResult{}, false, err
	}
	if phase.Kind != PhaseVote {
		return EndResult{}, false, nil
	}
	tally := tallyVotes(state, state.PhaseRoundKey)
	highest := 0
	for _, count := range tally {
		if count > highest {
			highest = count
		}
	}
	accused := make([]string, 0)
	if highest > 0 {
		for playerID, count := range tally {
			if count == highest {
				accused = append(accused, playerID)
			}
		}
	}
	if len(accused) != 1 {
		return EndResult{
			Completed:        true,
			ResultCode:       "no_elimination",
			WinnerTeam:       "culprit",
			AccusedPlayerIDs: accused,
			Summary: map[string]any{
				"tie_policy": config.Policy.TiePolicy,
				"tally":      tally,
				"accused":    accused,
			},
		}, true, nil
	}
	accusedRole, ok := roleForPlayer(state, accused[0])
	if !ok {
		return EndResult{}, false, errors.New("accused player has no role")
	}
	culprit := containsString(config.Policy.CulpritCharacterKeys, accusedRole.CharacterKey)
	winner := "culprit"
	resultCode := "wrong_accusation"
	if culprit {
		winner = "innocents"
		resultCode = "murder_solved"
	}
	return EndResult{
		Completed:        true,
		ResultCode:       resultCode,
		WinnerTeam:       winner,
		CulpritPlayerIDs: culpritPlayerIDs(state, config),
		AccusedPlayerIDs: accused,
		Summary: map[string]any{
			"tally":        tally,
			"accused":      accused,
			"culprit_keys": config.Policy.CulpritCharacterKeys,
			"winner_team":  winner,
		},
	}, true, nil
}

func resolveSearch(
	state RuntimeState,
	config ScriptConfig,
	playerID string,
	pointID string,
	now time.Time,
) (DecisionResult, DomainEvent, error) {
	point, ok := findSearchPoint(config, pointID)
	if !ok {
		return DecisionResult{}, DomainEvent{}, errors.New("search point does not exist")
	}
	candidates := make([]SearchPointClue, 0)
	for _, relation := range config.PointClues {
		if relation.SearchPointID == pointID {
			candidates = append(candidates, relation)
		}
	}
	if len(candidates) == 0 {
		return DecisionResult{}, DomainEvent{}, errors.New("search point has no clues")
	}
	var selected *Clue
	var relation SearchPointClue
	for _, candidate := range candidates {
		if candidate.IsDefault {
			relation = candidate
			break
		}
	}
	if relation.ClueID == "" {
		relation = candidates[0]
	}
	for index := range config.Clues {
		if config.Clues[index].ID == relation.ClueID {
			selected = &config.Clues[index]
			break
		}
	}
	if selected == nil {
		return DecisionResult{}, DomainEvent{}, errors.New("search point clue does not exist")
	}

	claims := 0
	for _, found := range state.FoundClues {
		if found.ClueID == selected.ID && found.ClaimSlot > 0 {
			claims++
		}
	}
	maxClaims := 1
	if selected.IsRepeatable {
		if selected.MaxDiscoveries != nil {
			maxClaims = *selected.MaxDiscoveries
		} else {
			maxClaims = int(^uint(0) >> 1)
		}
	}
	if claims >= maxClaims {
		return DecisionResult{}, DomainEvent{}, errors.New("clue is already claimed")
	}

	quotaKey := "quota:" + playerID + ":" + point.PhaseKey
	state.SearchPointUses[quotaKey]++
	state.SearchPointUses[pointID]++
	state.FoundClues = append(state.FoundClues, FoundClue{
		ClueID:           selected.ID,
		SearchPointID:    pointID,
		ClaimantPlayerID: playerID,
		ClaimSlot:        claims + 1,
		VisibilityScope:  selected.VisibilityScope,
		FoundAt:          now,
	})
	event := DomainEvent{
		EventType:       "game.clue_found",
		EventCategory:   "action",
		VisibilityScope: "player",
		Visibility:      map[string]any{"players": []string{playerID}},
		Sensitivity:     selected.Sensitivity,
		Payload: map[string]any{
			"schema_version":   "1.0.0",
			"clue_id":          selected.ID,
			"search_point_id":  pointID,
			"claim_slot":       claims + 1,
			"visibility_scope": selected.VisibilityScope,
		},
		StateChanged: true,
	}
	return DecisionResult{
		State:      state,
		ResultCode: "search_succeeded",
		ResultSummary: map[string]any{
			"clue_id":         selected.ID,
			"search_point_id": pointID,
			"claim_slot":      claims + 1,
		},
	}, event, nil
}

func actionEvent(eventType string, actor Actor, payload map[string]any) DomainEvent {
	return DomainEvent{
		EventType:       eventType,
		EventCategory:   "action",
		VisibilityScope: "room",
		Sensitivity:     0,
		Payload:         payload,
		StateChanged:    true,
	}
}

func systemEvent(eventType string, payload map[string]any) DomainEvent {
	payload["schema_version"] = "1.0.0"
	return DomainEvent{
		EventType:       eventType,
		EventCategory:   "system",
		VisibilityScope: "room",
		Sensitivity:     0,
		Payload:         payload,
		StateChanged:    true,
	}
}

func findActor(state RuntimeState, actor Actor) (Player, Role, bool) {
	player, playerFound := findPlayer(state, actor.PlayerID)
	role, roleFound := roleForPlayer(state, actor.PlayerID)
	return player, role, playerFound && roleFound
}

func findPlayer(state RuntimeState, playerID string) (Player, bool) {
	for _, player := range state.Players {
		if player.ID == playerID {
			return player, true
		}
	}
	return Player{}, false
}

func roleForPlayer(state RuntimeState, playerID string) (Role, bool) {
	for _, role := range state.Roles {
		if role.PlayerID == playerID {
			return role, true
		}
	}
	return Role{}, false
}

func findSearchPoint(config ScriptConfig, pointID string) (SearchPoint, bool) {
	for _, point := range config.SearchPoints {
		if point.ID == pointID {
			return point, true
		}
	}
	return SearchPoint{}, false
}

func containsAction(values []ActionType, value ActionType) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func setReadingCompleted(state *RuntimeState, playerID string) {
	for index := range state.Players {
		if state.Players[index].ID == playerID {
			state.Players[index].ReadingCompleted = true
		}
	}
}

func allPlayersReadingCompleted(state RuntimeState) bool {
	for _, player := range state.Players {
		if player.RuntimeStatus == "active" && !player.ReadingCompleted {
			return false
		}
	}
	return true
}

func allSearchQuotasExhausted(state RuntimeState, config ScriptConfig, phaseKey string) bool {
	for _, player := range state.Players {
		if player.RuntimeStatus != "active" {
			continue
		}
		if state.SearchPointUses["quota:"+player.ID+":"+phaseKey] < config.Policy.DefaultSearchQuota {
			return false
		}
	}
	return true
}

func hasVote(state RuntimeState, roundKey string, voterID string) bool {
	for _, vote := range state.Votes {
		if vote.RoundKey == roundKey && vote.VoterPlayerID == voterID {
			return true
		}
	}
	return false
}

func upsertVote(state *RuntimeState, vote Vote) {
	for index := range state.Votes {
		if state.Votes[index].RoundKey == vote.RoundKey &&
			state.Votes[index].VoterPlayerID == vote.VoterPlayerID {
			state.Votes[index] = vote
			return
		}
	}
	state.Votes = append(state.Votes, vote)
}

func allEligibleVotesSubmitted(state RuntimeState, config ScriptConfig) bool {
	for _, player := range state.Players {
		if player.RuntimeStatus != "active" {
			continue
		}
		role, ok := roleForPlayer(state, player.ID)
		if !ok || !containsString(config.Policy.VotingCharacterKeys, role.CharacterKey) {
			continue
		}
		if !hasVote(state, state.PhaseRoundKey, player.ID) {
			return false
		}
	}
	return true
}

func tallyVotes(state RuntimeState, roundKey string) map[string]int {
	tally := make(map[string]int)
	for _, vote := range state.Votes {
		if vote.RoundKey == roundKey && !vote.Abstain && vote.TargetPlayerID != "" {
			tally[vote.TargetPlayerID]++
		}
	}
	return tally
}

func culpritPlayerIDs(state RuntimeState, config ScriptConfig) []string {
	result := make([]string, 0)
	for _, role := range state.Roles {
		if containsString(config.Policy.CulpritCharacterKeys, role.CharacterKey) {
			result = append(result, role.PlayerID)
		}
	}
	return result
}

func cloneState(state RuntimeState) RuntimeState {
	cloned := state
	cloned.Players = append([]Player(nil), state.Players...)
	for index := range cloned.Players {
		cloned.Players[index].SearchQuotaByPhase =
			cloneIntMap(state.Players[index].SearchQuotaByPhase)
	}
	cloned.Roles = append([]Role(nil), state.Roles...)
	cloned.FoundClues = append([]FoundClue(nil), state.FoundClues...)
	cloned.Votes = append([]Vote(nil), state.Votes...)
	cloned.SearchPointUses = cloneIntMap(state.SearchPointUses)
	return cloned
}

func cloneIntMap(value map[string]int) map[string]int {
	if value == nil {
		return make(map[string]int)
	}
	cloned := make(map[string]int, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func EqualHash(left []byte, right []byte) bool {
	return hmac.Equal(left, right)
}

func StateHash(state RuntimeState) []byte {
	raw, _ := jsonMarshalCanonical(state)
	sum := sha256.Sum256(raw)
	return sum[:]
}

func jsonMarshalCanonical(value any) ([]byte, error) {
	return json.Marshal(value)
}
