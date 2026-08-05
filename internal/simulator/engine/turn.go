package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func validatePhaseCompletion(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	expectedPhase model.Phase,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != expectedPhase {
		return fmt.Errorf("game phase %q not currently in %q phase", state.Turn.Phase, expectedPhase)
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("%q is not the current active player %q", actingPlayerID, state.Turn.ActivePlayer)
	}
	if state.Turn.Number < 1 {
		return fmt.Errorf("turn count cannot be less than 1: %d", state.Turn.Number)
	}
	return nil
}

func completeRecoveryPhase(state *model.MatchState, actingPlayerID model.PlayerID) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseRecovery); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	if state.Turn.Number == 1 {
		if state.Turn.ActivePlayer != state.FirstPlayer {
			return fmt.Errorf("active player %q is not set as the first player", state.Turn.ActivePlayer)
		}
		state.Turn.Phase = model.PhaseCall
	} else {
		if err := enterDrawPhase(state); err != nil {
			return fmt.Errorf("error transitioning to draw phase: %w", err)
		}
	}
	return nil
}

func enterDrawPhase(state *model.MatchState) error {
	if err := validatePhaseCompletion(state, state.Turn.ActivePlayer, model.PhaseRecovery); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	if state.Turn.Number == 1 {
		return fmt.Errorf("draw phase cannot be entered during first turn")
	}
	var activeIndex int = -1
	for index := range state.Players {
		if state.Players[index].ID == state.Turn.ActivePlayer {
			activeIndex = index
		}
	}
	if activeIndex == -1 {
		return fmt.Errorf("player ID not found")
	}
	if len(state.Players[activeIndex].Deck) == 0 {
		return fmt.Errorf("deck cannot be empty")
	}
	card := state.Players[activeIndex].Deck[0]
	state.Players[activeIndex].Deck = slices.Delete(state.Players[activeIndex].Deck, 0, 1)
	state.Players[activeIndex].Hand = append(state.Players[activeIndex].Hand, card)
	state.Turn.Phase = model.PhaseDraw
	return nil
}

func completeDrawPhase(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseDraw); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	if state.Turn.Number == 1 {
		return fmt.Errorf("first turn cannot be entered on turn 1")
	}
	state.Turn.Phase = model.PhaseCall
	return nil
}

func completeCallPhase(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseCall); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	state.Turn.Phase = model.PhaseMain
	return nil
}

func completeMainPhase(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseMain); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	state.Turn.Phase = model.PhaseBattle
	return nil
}

func completeBattlePhase(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseBattle); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	state.Turn.Phase = model.PhaseEnd
	return nil
}

func completeEndPhase(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) error {
	if err := validatePhaseCompletion(state, actingPlayerID, model.PhaseEnd); err != nil {
		return fmt.Errorf("error validating phase: %w", err)
	}
	actingIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingIndex = index
		}
	}
	if actingIndex == -1 {
		return fmt.Errorf("player %q not found in players", actingPlayerID)
	}
	incomingIndex := 1 - actingIndex
	incomingPlayerID := state.Players[incomingIndex].ID
	if err := recoverPlayerCards(state, incomingPlayerID); err != nil {
		return fmt.Errorf("error recovering cards: %w", err)
	}
	state.Turn.ActivePlayer = incomingPlayerID
	for index := range state.Players {
		state.Players[index].Aether = model.AetherPool{}
	}
	state.Turn.Number++
	state.Turn.Phase = model.PhaseRecovery
	state.Turn.CallActionTaken = false

	return nil
}

func CompleteCurrentPhase(state *model.MatchState, actingPlayerID model.PlayerID, expectedRevision model.Revision) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	switch state.Turn.Phase {
	case model.PhaseRecovery:
		if err := completeRecoveryPhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	case model.PhaseDraw:
		if err := completeDrawPhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	case model.PhaseCall:
		if err := completeCallPhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	case model.PhaseMain:
		if err := completeMainPhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	case model.PhaseBattle:
		if err := completeBattlePhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	case model.PhaseEnd:
		if err := completeEndPhase(state, actingPlayerID); err != nil {
			return fmt.Errorf("transition from %q: %w", state.Turn.Phase, err)
		}
	default:
		return fmt.Errorf("unsupported or illegal transition from %q", state.Turn.Phase)
	}
	state.Revision++
	return nil
}
