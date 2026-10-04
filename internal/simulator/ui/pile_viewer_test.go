package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestGraveyardPileOpensOnActivate(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Graveyard = []simulatorview.CardView{
		{MatchID: "g1", CardID: "visible-servant-card", ShowFace: true, Owner: "player-one"},
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		MoveCard: func(model.MoveCardCommand, model.Revision) {},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	viewer := screen.playerBoards[1]
	layer := findZoneInteractLayer(viewer.graveyard)
	if layer == nil || layer.onTap == nil {
		t.Fatal("graveyard missing left-click browse layer")
	}
	// Opening the dialog is enough to prove the wiring; Close immediately.
	layer.onTap()
}

func TestBrowseDeckUsesProjectedDeckCards(t *testing.T) {
	match := testMatchView()
	match.Players[0].Deck = []simulatorview.CardView{
		{MatchID: "d1", CardID: "visible-hand-card", Owner: "player-one"},
		{MatchID: "d2", CardID: "visible-servant-card", Owner: "player-one"},
	}
	match.Players[0].DeckCount = 2
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		MoveCard: func(model.MoveCardCommand, model.Revision) {},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	viewer := screen.playerBoards[1]
	if len(viewer.player.Deck) != 2 {
		t.Fatalf("viewer deck projection len = %d", len(viewer.player.Deck))
	}
	viewer.beginBrowseDeck()
}

func TestClosingDeckBrowseShufflesDeck(t *testing.T) {
	var shuffledRevision model.Revision
	var shuffleCalls int
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 7
	match.Players[0].Deck = []simulatorview.CardView{
		{MatchID: "d1", CardID: "visible-hand-card", Owner: "player-one"},
	}
	match.Players[0].DeckCount = 1
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		ShuffleDeck: func(revision model.Revision) {
			shuffleCalls++
			shuffledRevision = revision
		},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	popup := screen.showPileViewer(pileViewerRequest{
		title:    "Player Deck",
		zone:     model.ZoneDeck,
		playerID: match.Players[0].ID,
		cards:    match.Players[0].Deck,
		canMove:  false,
	})
	if popup == nil {
		t.Fatal("expected deck browse dialog")
	}
	popup.Hide()
	if shuffleCalls != 1 {
		t.Fatalf("ShuffleDeck calls = %d; want 1", shuffleCalls)
	}
	if shuffledRevision != match.Revision {
		t.Fatalf("ShuffleDeck revision = %d; want %d", shuffledRevision, match.Revision)
	}
}
