package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestDragDropMovesCardToGraveyard(t *testing.T) {
	var got model.MoveCardCommand
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "hand-1", CardID: "visible-hand-card", Owner: "player-one", ShowFace: true},
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		MoveCard: func(command model.MoveCardCommand, revision model.Revision) {
			got = command
		},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()
	window.Resize(fyne.NewSize(1200, 900))

	viewer := screen.playerBoards[1]
	tiles := collectCardTiles(viewer.hand)
	if len(tiles) == 0 {
		t.Fatal("expected hand tiles")
	}
	tile := tiles[0]
	if tile.OnDragBegin == nil || tile.OnDragEnd == nil {
		t.Fatal("hand card is not draggable")
	}

	screen.beginCardDrag(tile.View, model.ZoneHand, match.Players[0].ID)
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(viewer.graveyard)
	size := viewer.graveyard.Size()
	screen.trackCardDrag(fyne.NewPos(origin.X+size.Width/2, origin.Y+size.Height/2))
	screen.finishCardDrag()

	if got.CardID != "hand-1" || got.DestinationZone != model.ZoneGraveyard || got.DestinationFace != model.CardFaceUp {
		t.Fatalf("MoveCard = %#v", got)
	}
}
