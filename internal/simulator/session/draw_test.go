package session

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestPlayerSessionDrawCardsUsesBoundPlayerAndPrivateViews(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Players[0].Deck = []model.MatchCardID{"p1-deck-top", "p1-deck-next"}
	state.Players[0].Hand = []model.MatchCardID{"p1-hand"}
	state.CardInstances["p1-deck-top"] = model.CardInstance{
		MatchID: "p1-deck-top", CardID: "printed", Owner: "player-one", Controller: "player-one",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.CardInstances["p1-deck-next"] = model.CardInstance{
		MatchID: "p1-deck-next", CardID: "printed", Owner: "player-one", Controller: "player-one",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.CardInstances["p1-hand"] = model.CardInstance{
		MatchID: "p1-hand", CardID: "printed", Owner: "player-one", Controller: "player-one",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	match, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
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

	projection, err := one.DrawCards(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Revision != 1 || projection.Players[0].DeckCount != 1 {
		t.Fatalf("projection revision/deck = %d/%d; want 1/1", projection.Revision, projection.Players[0].DeckCount)
	}
	if len(projection.Players[0].Hand) != 2 || projection.Players[0].Hand[1].MatchID != "p1-deck-top" {
		t.Fatalf("drawer hand = %#v; want drawn card visible", projection.Players[0].Hand)
	}

	opponentView, err := two.View()
	if err != nil {
		t.Fatal(err)
	}
	if opponentView.Players[0].DeckCount != 1 {
		t.Fatalf("opponent deck count = %d; want 1", opponentView.Players[0].DeckCount)
	}
	hiddenHand := opponentView.Players[0].Hand
	if len(hiddenHand) != 2 || hiddenHand[1].MatchID != "" || hiddenHand[1].CardID != "" {
		t.Fatalf("opponent saw draw identities: %#v", hiddenHand)
	}

	if _, err := one.DrawCards(1, 0); err == nil {
		t.Fatal("stale draw accepted")
	}
	if match.state.Revision != 1 {
		t.Fatal("rejected draw changed revision")
	}
}
