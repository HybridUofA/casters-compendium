package session

import (
	"reflect"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestPlayerSessionPeekDeckTopsPrivateAndNonMutating(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Players[1].Deck = []model.MatchCardID{"p2-d1", "p2-d2", "p2-d3"}
	state.CardInstances["p2-d1"] = model.CardInstance{
		MatchID: "p2-d1", CardID: "enemy-top", Owner: "player-two", Controller: "player-two",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.CardInstances["p2-d2"] = model.CardInstance{
		MatchID: "p2-d2", CardID: "enemy-next", Owner: "player-two", Controller: "player-two",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.CardInstances["p2-d3"] = model.CardInstance{
		MatchID: "p2-d3", CardID: "enemy-third", Owner: "player-two", Controller: "player-two",
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

	peeked, projection, err := one.PeekDeckTops("player-two", 1)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Revision != 0 {
		t.Fatalf("peek bumped revision to %d", projection.Revision)
	}
	if len(peeked) != 1 || peeked[0].MatchID != "p2-d1" || peeked[0].CardID != "enemy-top" {
		t.Fatalf("peeked = %#v; want enemy top identity", peeked)
	}
	if !peeked[0].ShowFace {
		t.Fatal("peek should reveal face to actor")
	}

	ownerView, err := two.View()
	if err != nil {
		t.Fatal(err)
	}
	if ownerView.Revision != 0 {
		t.Fatalf("owner revision = %d; want 0", ownerView.Revision)
	}
	if ownerView.Players[1].DeckCount != 3 {
		t.Fatalf("deck count = %d; want 3", ownerView.Players[1].DeckCount)
	}
}

func TestPlayerSessionMoveDeckTopToBottomAndDig(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Players[0].Deck = []model.MatchCardID{"p1-d1", "p1-d2", "p1-d3"}
	state.Players[0].Hand = nil
	state.Players[1].Deck = []model.MatchCardID{"p2-d1", "p2-d2"}
	for _, id := range []model.MatchCardID{"p1-d1", "p1-d2", "p1-d3", "p2-d1", "p2-d2"} {
		state.CardInstances[id] = model.CardInstance{
			MatchID: id, CardID: "printed", Owner: ownerForMatchCard(id), Controller: ownerForMatchCard(id),
			CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
		}
	}
	match, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatal(err)
	}
	one, err := NewPlayerSession(match, "player-one")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := one.MoveDeckTopToBottom("player-two", 0); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(match.state.Players[1].Deck, []model.MatchCardID{"p2-d2", "p2-d1"}) {
		t.Fatalf("enemy deck after bottom = %#v", match.state.Players[1].Deck)
	}

	if _, err := one.ResolveDeckDig(
		"p1-d2",
		[]model.MatchCardID{"p1-d3", "p1-d1"},
		1,
	); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(match.state.Players[0].Hand, []model.MatchCardID{"p1-d2"}) {
		t.Fatalf("hand = %#v; want kept dig card", match.state.Players[0].Hand)
	}
	if !reflect.DeepEqual(match.state.Players[0].Deck, []model.MatchCardID{"p1-d3", "p1-d1"}) {
		t.Fatalf("deck = %#v; want bottomed dig remainder", match.state.Players[0].Deck)
	}
	if match.state.Revision != 2 {
		t.Fatalf("revision = %d; want 2", match.state.Revision)
	}
}

func ownerForMatchCard(id model.MatchCardID) model.PlayerID {
	if len(id) >= 2 && id[:2] == "p2" {
		return "player-two"
	}
	return "player-one"
}
