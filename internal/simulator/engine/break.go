package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func clearPendingBreak(state *model.MatchState) {
	if state == nil {
		return
	}
	state.PendingBreak = model.PendingBreak{}
}

func queueBreakIfPresent(
	state *model.MatchState,
	catalog rules.CardCatalog,
	ownerID model.PlayerID,
	cardIDs ...model.MatchCardID,
) {
	if state == nil || catalog == nil || len(cardIDs) == 0 {
		return
	}
	queued := make([]model.MatchCardID, 0, len(cardIDs))
	for _, cardID := range cardIDs {
		instance, ok := state.CardInstances[cardID]
		if !ok {
			continue
		}
		definition, found := catalog.FindByID(string(instance.CardID))
		if !found || !cardDeclaresBreak(definition.Ability) {
			continue
		}
		queued = append(queued, cardID)
	}
	if len(queued) == 0 {
		return
	}
	state.PendingBreak = model.PendingBreak{
		PlayerID: ownerID,
		CardIDs:  queued,
	}
}

func finishBreakOrReopen(state *model.MatchState) {
	if len(state.PendingBreak.CardIDs) > 0 {
		return
	}
	clearPendingBreak(state)
	reopenPriorityForActivePlayer(state)
}

// DeclineBreak skips the optional Break for the currently offered corrupted card.
func DeclineBreak(
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
	if state.PendingBreak.PlayerID == "" || len(state.PendingBreak.CardIDs) == 0 {
		return fmt.Errorf("no break decision is pending")
	}
	if actingPlayerID != state.PendingBreak.PlayerID {
		return fmt.Errorf("only the corrupted Orb's owner may decline break")
	}
	state.PendingBreak.CardIDs = state.PendingBreak.CardIDs[1:]
	finishBreakOrReopen(state)
	state.Revision++
	return nil
}

// PlayBreak plays the offered Break card immediately without paying its cost.
// Printed effects remain manual after the card enters chase / resolves.
func PlayBreak(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	entryOrientation model.CardOrientation,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if catalog == nil {
		return fmt.Errorf("catalog is required")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if state.PendingBreak.PlayerID == "" || len(state.PendingBreak.CardIDs) == 0 {
		return fmt.Errorf("no break decision is pending")
	}
	if actingPlayerID != state.PendingBreak.PlayerID {
		return fmt.Errorf("only the corrupted Orb's owner may play break")
	}
	if cardID == "" {
		cardID = state.PendingBreak.CardIDs[0]
	}
	if state.PendingBreak.CardIDs[0] != cardID {
		return fmt.Errorf("break must be decided for %q first", state.PendingBreak.CardIDs[0])
	}

	playerIndex, err := findPlayerIndex(state, actingPlayerID)
	if err != nil {
		return err
	}
	handIndex := slices.Index(state.Players[playerIndex].Hand, cardID)
	if handIndex < 0 {
		return fmt.Errorf("break card %q is not in hand", cardID)
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("break card %q was not found", cardID)
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return fmt.Errorf("break card definition %q was not found", instance.CardID)
	}
	if !cardDeclaresBreak(definition.Ability) {
		return fmt.Errorf("card does not declare Break")
	}

	cardKind := strings.ToLower(strings.TrimSpace(definition.Type))
	switch cardKind {
	case "servant":
		if entryOrientation == "" {
			entryOrientation = model.OrientationRecovered
		}
		if entryOrientation != model.OrientationRecovered && entryOrientation != model.OrientationReversed {
			return fmt.Errorf("servant break orientation must be Recovered or Reversed")
		}
	case "conjure", "barrier":
		entryOrientation = ""
	default:
		return fmt.Errorf("cannot play break for card type %q", definition.Type)
	}

	if state.NextLinkID == 0 {
		state.NextLinkID = 1
	}

	state.Players[playerIndex].Hand = slices.Delete(state.Players[playerIndex].Hand, handIndex, handIndex+1)
	instance.Face = model.CardFaceUp
	instance.Controller = actingPlayerID
	state.CardInstances[cardID] = instance
	state.ChaseLinks = append(state.ChaseLinks, model.ChaseLink{
		ID:               state.NextLinkID,
		Controller:       actingPlayerID,
		SourceCardID:     cardID,
		Kind:             model.ChaseLinkCardPlay,
		EntryOrientation: entryOrientation,
	})
	state.NextLinkID++
	state.PendingBreak.CardIDs = state.PendingBreak.CardIDs[1:]
	if len(state.PendingBreak.CardIDs) == 0 {
		clearPendingBreak(state)
	}
	// Break play opens a priority sequence for responses before resolution.
	state.PrioritySequenceOpen = true
	state.PriorityHolder = actingPlayerID
	state.PassCount = 0
	state.Revision++
	return nil
}
