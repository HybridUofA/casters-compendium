package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

const sageAdviceCardName = "Sage Advice"

func controlsSageAdvice(
	state *model.MatchState,
	catalog rules.CardCatalog,
	playerID model.PlayerID,
) bool {
	if state == nil || catalog == nil {
		return false
	}
	playerIndex, err := findPlayerIndex(state, playerID)
	if err != nil {
		return false
	}
	for _, matchID := range state.Players[playerIndex].ServantZone {
		instance, ok := state.CardInstances[matchID]
		if !ok || instance.Controller != playerID {
			continue
		}
		if instance.Face == model.CardFaceDown {
			continue
		}
		definition, found := catalog.FindByID(string(instance.CardID))
		if !found {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(definition.Name), sageAdviceCardName) &&
			strings.EqualFold(strings.TrimSpace(definition.Type), "Barrier") {
			return true
		}
	}
	return false
}

func clearPendingDraw(state *model.MatchState) {
	state.PendingDraw = model.PendingDraw{}
}

// beginDrawOrOfferReplacement draws count cards, or pauses for Sage Advice on
// each card when the drawer controls that Barrier.
func beginDrawOrOfferReplacement(
	state *model.MatchState,
	catalog rules.CardCatalog,
	playerID model.PlayerID,
	count int,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if count < 1 {
		return fmt.Errorf("draw count must be at least 1")
	}
	if state.PendingDraw.Step != model.PendingDrawIdle {
		return fmt.Errorf("a draw replacement is already pending")
	}
	if !controlsSageAdvice(state, catalog, playerID) {
		return DrawCards(state, playerID, count)
	}
	state.PendingDraw = model.PendingDraw{
		PlayerID:  playerID,
		Remaining: count,
		Step:      model.PendingDrawOffer,
	}
	return nil
}

// DeclineDrawReplacement draws the next pending card normally, then continues
// any remaining draws (offering Sage Advice again when still applicable).
func DeclineDrawReplacement(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if state.PendingDraw.Step != model.PendingDrawOffer {
		return fmt.Errorf("no draw replacement offer to decline")
	}
	if actingPlayerID != state.PendingDraw.PlayerID {
		return fmt.Errorf("only the drawing player may decline the replacement")
	}
	if err := DrawCards(state, actingPlayerID, 1); err != nil {
		return err
	}
	return continuePendingDraws(state, catalog)
}

// AcceptSageAdvice begins the dig replacement for the next pending draw.
func AcceptSageAdvice(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d, got %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if state.PendingDraw.Step != model.PendingDrawOffer {
		return fmt.Errorf("no draw replacement offer to accept")
	}
	if actingPlayerID != state.PendingDraw.PlayerID {
		return fmt.Errorf("only the drawing player may accept Sage Advice")
	}
	playerIndex, err := findPlayerIndex(state, actingPlayerID)
	if err != nil {
		return err
	}
	if len(state.Players[playerIndex].Deck) == 0 {
		return fmt.Errorf("deck is empty")
	}
	state.PendingDraw.Step = model.PendingDrawDig
	state.Revision++
	return nil
}

func continuePendingDraws(state *model.MatchState, catalog rules.CardCatalog) error {
	shouldReopen := !state.PrioritySequenceOpen
	if state.PendingDraw.Remaining <= 1 {
		clearPendingDraw(state)
		if shouldReopen {
			reopenPriorityForActivePlayer(state)
		}
		state.Revision++
		return nil
	}
	state.PendingDraw.Remaining--
	if controlsSageAdvice(state, catalog, state.PendingDraw.PlayerID) {
		state.PendingDraw.Step = model.PendingDrawOffer
		state.Revision++
		return nil
	}
	remaining := state.PendingDraw.Remaining
	playerID := state.PendingDraw.PlayerID
	clearPendingDraw(state)
	if err := DrawCards(state, playerID, remaining); err != nil {
		return err
	}
	if shouldReopen {
		reopenPriorityForActivePlayer(state)
	}
	state.Revision++
	return nil
}

func finishPendingDig(state *model.MatchState, catalog rules.CardCatalog) error {
	if state.PendingDraw.Step != model.PendingDrawDig {
		return fmt.Errorf("no Sage Advice dig is pending")
	}
	return continuePendingDraws(state, catalog)
}
