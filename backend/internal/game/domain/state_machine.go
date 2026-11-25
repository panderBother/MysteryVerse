package domain

import (
	"errors"
	"fmt"
	"time"
)

type StateMachine struct {
	phases []Phase
}

func NewStateMachine(phases []Phase) (*StateMachine, error) {
	if len(phases) == 0 {
		return nil, errors.New("script must define at least one phase")
	}
	seen := make(map[string]bool, len(phases))
	for index, phase := range phases {
		if phase.Key == "" {
			return nil, fmt.Errorf("phase %d has an empty key", index)
		}
		if seen[phase.Key] {
			return nil, fmt.Errorf("duplicate phase key %q", phase.Key)
		}
		seen[phase.Key] = true
		if err := phase.Kind.Validate(); err != nil {
			return nil, err
		}
		if index > 0 && phases[index-1].Sequence >= phase.Sequence {
			return nil, errors.New("phases must be ordered by sequence")
		}
	}
	return &StateMachine{phases: append([]Phase(nil), phases...)}, nil
}

func (m *StateMachine) Current(state RuntimeState) (Phase, error) {
	if state.PhaseIndex < 0 || state.PhaseIndex >= len(m.phases) {
		return Phase{}, errors.New("runtime phase index is out of range")
	}
	return m.phases[state.PhaseIndex], nil
}

func (m *StateMachine) Next(state RuntimeState) (Phase, bool, error) {
	next := state.PhaseIndex + 1
	if next >= len(m.phases) {
		return Phase{}, false, nil
	}
	return m.phases[next], true, nil
}

func (m *StateMachine) Enter(
	state RuntimeState,
	index int,
	now time.Time,
) (RuntimeState, DomainEvent, error) {
	if index < 0 || index >= len(m.phases) {
		return RuntimeState{}, DomainEvent{}, errors.New("phase index is out of range")
	}
	phase := m.phases[index]
	state.PhaseIndex = index
	state.PhaseKey = phase.Key
	state.PhaseStartedAt = now
	state.PhaseRoundKey = phase.Key
	if phase.TimeoutSeconds != nil && *phase.TimeoutSeconds > 0 {
		deadline := now.Add(time.Duration(*phase.TimeoutSeconds) * time.Second)
		state.PhaseDeadlineAt = &deadline
	} else {
		state.PhaseDeadlineAt = nil
	}
	return state, DomainEvent{
		EventType:       "game.phase_started",
		EventCategory:   "phase",
		VisibilityScope: "room",
		Sensitivity:     0,
		Payload: map[string]any{
			"schema_version": "1.0.0",
			"phase_key":      phase.Key,
			"phase_type":     phase.Kind,
			"sequence_no":    phase.Sequence,
		},
		StateChanged: true,
	}, nil
}

func (m *StateMachine) Advance(
	state RuntimeState,
	now time.Time,
) (RuntimeState, []DomainEvent, bool, error) {
	_, exists, err := m.Next(state)
	if err != nil {
		return state, nil, false, err
	}
	if !exists {
		return state, nil, false, nil
	}
	state, event, err := m.Enter(state, state.PhaseIndex+1, now)
	if err != nil {
		return state, nil, false, err
	}
	return state, []DomainEvent{event}, true, nil
}

func (m *StateMachine) CanTransition(current Status, target Status) bool {
	return current.CanTransitionTo(target)
}
