package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestOpponentDeckOffersPeekWhenWired(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[1].DeckCount = 4
	var peekedOwner model.PlayerID
	var peekedCount int
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			PeekDeckTops: func(ownerID model.PlayerID, count int, done func([]simulatorview.CardView, error)) {
				peekedOwner = ownerID
				peekedCount = count
				if done != nil {
					done([]simulatorview.CardView{{
						MatchID:  "enemy-top",
						CardID:   "definition",
						ShowFace: true,
					}}, nil)
				}
			},
		},
		nil,
	)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	opponentBoard := screen.playerBoards[0]
	if findZoneInteractLayer(opponentBoard.deck) == nil {
		t.Fatal("opponent deck should offer peek context actions")
	}
	opponentBoard.beginLookAtEnemyTop()
	if peekedOwner != opponentBoard.player.ID || peekedCount != 1 {
		t.Fatalf("peek owner/count = %s/%d; want %s/1", peekedOwner, peekedCount, opponentBoard.player.ID)
	}
}

func TestViewerDeckOffersSageDigWhenWired(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[0].DeckCount = 5
	var peekedOwner model.PlayerID
	var peekedCount int
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			PeekDeckTops: func(ownerID model.PlayerID, count int, done func([]simulatorview.CardView, error)) {
				peekedOwner = ownerID
				peekedCount = count
				if done != nil {
					done([]simulatorview.CardView{
						{MatchID: "a", CardID: "definition", ShowFace: true},
						{MatchID: "b", CardID: "definition", ShowFace: true},
						{MatchID: "c", CardID: "definition", ShowFace: true},
					}, nil)
				}
			},
			ResolveDeckDig: func(model.MatchCardID, []model.MatchCardID, model.Revision) {},
		},
		nil,
	)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	viewerBoard := screen.playerBoards[1]
	if findZoneInteractLayer(viewerBoard.deck) == nil {
		t.Fatal("viewer deck should offer dig context actions")
	}
	viewerBoard.beginSageDig()
	if peekedOwner != viewerBoard.player.ID || peekedCount != 3 {
		t.Fatalf("dig peek owner/count = %s/%d; want %s/3", peekedOwner, peekedCount, viewerBoard.player.ID)
	}
}
