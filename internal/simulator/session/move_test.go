package session

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestPlayerSessionMoveCardUsesBoundPlayerAndPrivateViews(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Players[0].ServantZone = []model.MatchCardID{"stolen"}
	state.CardInstances["stolen"] = model.CardInstance{MatchID: "stolen", CardID: "servant", Owner: "player-two", Controller: "player-one", CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRested}
	match, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{"servant": gamecards.Card{ID: "servant", Type: "Servant"}})
	if err != nil {
		t.Fatal(err)
	}
	one, err := NewPlayerSession(match, "player-one")
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewPlayerSession(match, "player-two")
	if err != nil {
		t.Fatal(err)
	}
	command := model.MoveCardCommand{CardID: "stolen", DestinationPlayerID: "player-two", DestinationZone: model.ZoneHand, DestinationFace: model.CardFaceDown}
	if _, err := two.MoveCard(command, 0); err == nil {
		t.Fatal("owner moved a card controlled by opponent")
	}
	projection, err := one.MoveCard(command, 0)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Revision != 1 || len(projection.Players[0].ServantZone) != 0 {
		t.Fatal("move not projected")
	}
	hand := projection.Players[1].Hand
	if hand[len(hand)-1].MatchID != "" || hand[len(hand)-1].CardID != "" {
		t.Fatal("opponent's hand identity leaked")
	}
	ownerView, err := two.View()
	if err != nil {
		t.Fatal(err)
	}
	ownerHand := ownerView.Players[1].Hand
	if ownerHand[len(ownerHand)-1].MatchID != "stolen" {
		t.Fatal("owner cannot see returned card")
	}
	command.DestinationZone = model.ZoneDeck
	command.Placement = model.DeckPlacementTop
	if _, err := two.MoveCard(command, 0); err == nil {
		t.Fatal("stale move accepted")
	}
	if match.state.Revision != 1 {
		t.Fatal("rejected move changed revision")
	}
	command.DestinationZone = model.ZoneExile
	command.Placement = ""
	if _, err := two.MoveCard(command, 1); err == nil {
		t.Fatal("unsupported hidden Exile accepted")
	}
	if match.state.Revision != 1 {
		t.Fatal("unsupported move changed revision")
	}
}
