package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

type castValidator func(
	*model.MatchState,
	rules.CardCatalog,
	model.PlayerID,
	model.MatchCardID,
	model.AetherPayment,
) error

func CastServant(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, entryOrientation model.CardOrientation, expectedRevision model.Revision) error {
	validate := func(
		state *model.MatchState,
		catalog rules.CardCatalog,
		actingPlayerID model.PlayerID,
		cardID model.MatchCardID,
		payment model.AetherPayment,
	) error {
		return rules.ValidateCastServant(
			state,
			catalog,
			actingPlayerID,
			cardID,
			payment,
			entryOrientation,
		)
	}
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"servant",
		validate,
		entryOrientation,
	)
}

func CastConjure(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) error {
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"conjure",
		rules.ValidateCastConjure,
		"",
	)
}

func CastBarrier(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) error {
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"barrier",
		rules.ValidateCastBarrier,
		"",
	)
}

func castCard(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	payment model.AetherPayment,
	expectedRevision model.Revision,
	cardKind string,
	validate castValidator,
	entryOrientation model.CardOrientation,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("expected revision %d, current revision is %d", expectedRevision, state.Revision)
	}
	err := validate(state, catalog, actingPlayerID, cardID, payment)
	if err != nil {
		return fmt.Errorf("error casting %s: %w", cardKind, err)
	}
	playerIndex := -1
	for index, player := range state.Players {
		if player.ID == actingPlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("active player not found")
	}
	nonActingPlayerID := state.Players[(1 - playerIndex)].ID
	if nonActingPlayerID == "" || nonActingPlayerID == actingPlayerID {
		return fmt.Errorf("opposing player was not found")
	}
	cardIndex := -1
	for index, ID := range state.Players[playerIndex].Hand {
		if ID == cardID {
			cardIndex = index
			break
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card %q not found in %q player's hand", cardID, actingPlayerID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q was not found in card instances", cardID)
	}
	payAether(&state.Players[playerIndex].Aether, payment)
	hand := state.Players[playerIndex].Hand
	hand = slices.Delete(hand, cardIndex, cardIndex+1)
	state.Players[playerIndex].Hand = hand
	instance.Face = model.CardFaceUp
	instance.Controller = actingPlayerID
	state.CardInstances[cardID] = instance
	link := model.ChaseLink{
		ID:               state.NextLinkID,
		Controller:       actingPlayerID,
		SourceCardID:     cardID,
		Kind:             model.ChaseLinkCardPlay,
		EntryOrientation: entryOrientation,
	}
	state.ChaseLinks = append(state.ChaseLinks, link)
	state.NextLinkID++
	state.PassCount = 0
	state.PriorityHolder = nonActingPlayerID
	state.Revision++
	return nil
}
