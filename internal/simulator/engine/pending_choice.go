package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func clearPendingChoice(state *model.MatchState) {
	if state == nil {
		return
	}
	state.PendingChoice = model.PendingChoice{}
}

// OfferPendingChoice opens a generic decision prompt for Hybrid-owned effects.
// It does not bump revision; the caller bumps when the opening effect commits.
func OfferPendingChoice(state *model.MatchState, choice model.PendingChoice) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if choice.PlayerID == "" {
		return fmt.Errorf("pending choice player ID cannot be empty")
	}
	if state.PendingChoice.PlayerID != "" {
		return fmt.Errorf("a pending choice is already open for %q", state.PendingChoice.PlayerID)
	}
	if state.PendingBreak.PlayerID != "" {
		return fmt.Errorf("cannot open pending choice while Break is pending")
	}
	if state.PendingDraw.Step != model.PendingDrawIdle {
		return fmt.Errorf("cannot open pending choice while a draw replacement is pending")
	}
	state.PendingChoice = model.PendingChoice{
		PlayerID:     choice.PlayerID,
		Kind:         choice.Kind,
		Prompt:       choice.Prompt,
		AllowDecline: choice.AllowDecline,
		SourceCardID: choice.SourceCardID,
		Payload:      choice.Payload,
	}
	if len(choice.Options) > 0 {
		state.PendingChoice.Options = append([]model.PendingChoiceOption(nil), choice.Options...)
	}
	if len(choice.SelectableCardIDs) > 0 {
		state.PendingChoice.SelectableCardIDs = append([]model.MatchCardID(nil), choice.SelectableCardIDs...)
	}
	return nil
}

// ResolvePendingChoice accepts the open PendingChoice and runs Kind-specific
// effect continuations when Kind is an effects pending key.
func ResolvePendingChoice(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	optionID string,
	selectedCardIDs []model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if state.PendingChoice.PlayerID == "" {
		return fmt.Errorf("no pending choice is open")
	}
	if actingPlayerID != state.PendingChoice.PlayerID {
		return fmt.Errorf("only %q may resolve the pending choice", state.PendingChoice.PlayerID)
	}
	if len(state.PendingChoice.Options) > 0 {
		if optionID == "" {
			return fmt.Errorf("pending choice requires an option ID")
		}
		found := false
		for _, option := range state.PendingChoice.Options {
			if option.ID == optionID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("option %q is not part of the pending choice", optionID)
		}
	} else if optionID != "" {
		return fmt.Errorf("pending choice has no options")
	}
	if len(state.PendingChoice.SelectableCardIDs) > 0 {
		if len(selectedCardIDs) == 0 {
			return fmt.Errorf("pending choice requires at least one selected card")
		}
		for _, selected := range selectedCardIDs {
			if !slices.Contains(state.PendingChoice.SelectableCardIDs, selected) {
				return fmt.Errorf("card %q is not a selectable target", selected)
			}
		}
	} else if len(selectedCardIDs) > 0 {
		return fmt.Errorf("pending choice has no selectable cards")
	}
	pending := state.PendingChoice
	clearPendingChoice(state)
	if err := effects.ContinuePendingChoice(pending, selectedCardIDs, optionID, effects.Context{
		State:      state,
		Source:     pending.SourceCardID,
		Controller: actingPlayerID,
		Catalog:    catalog,
		DestroyField: func(cardID model.MatchCardID) error {
			return destroyServantWithCause(state, catalog, cardID, destroyCauseEffect, actingPlayerID)
		},
		OfferChoice: func(choice model.PendingChoice) error {
			return OfferPendingChoice(state, choice)
		},
	}); err != nil {
		return err
	}
	// Nested OfferChoice (multi-step recipes) keeps priority paused.
	if state.PendingChoice.PlayerID == "" && state.MatchStatus == model.StatusInProgress {
		reopenPriorityForActivePlayer(state)
	}
	state.Revision++
	return nil
}

// DeclinePendingChoice skips an optional pending choice when AllowDecline is set.
func DeclinePendingChoice(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if state.PendingChoice.PlayerID == "" {
		return fmt.Errorf("no pending choice is open")
	}
	if actingPlayerID != state.PendingChoice.PlayerID {
		return fmt.Errorf("only %q may decline the pending choice", state.PendingChoice.PlayerID)
	}
	if !state.PendingChoice.AllowDecline {
		return fmt.Errorf("pending choice cannot be declined")
	}
	clearPendingChoice(state)
	if state.MatchStatus == model.StatusInProgress {
		reopenPriorityForActivePlayer(state)
	}
	state.Revision++
	return nil
}
