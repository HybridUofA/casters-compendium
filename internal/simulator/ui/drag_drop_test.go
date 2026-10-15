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

	screen.beginCardDrag(tile, model.ZoneHand, match.Players[0].ID)
	if screen.drag.ghost == nil {
		t.Fatal("expected drag ghost after begin")
	}
	if screen.dragLayer == nil || len(screen.dragLayer.Objects) != 1 {
		t.Fatalf("drag layer objects = %d; want ghost present", len(screen.dragLayer.Objects))
	}
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(viewer.graveyard)
	size := viewer.graveyard.Size()
	dropAt := fyne.NewPos(origin.X+size.Width/2, origin.Y+size.Height/2)
	screen.trackCardDrag(dropAt)
	ghostPos := screen.drag.ghost.Position()
	if ghostPos.X == 0 && ghostPos.Y == 0 {
		t.Fatal("expected ghost to move under pointer")
	}
	screen.finishCardDrag()
	if screen.drag.active || screen.drag.ghost != nil {
		t.Fatal("expected drag session cleared after finish")
	}
	if len(screen.dragLayer.Objects) != 0 {
		t.Fatalf("drag layer objects = %d; want empty after finish", len(screen.dragLayer.Objects))
	}

	if got.CardID != "hand-1" || got.DestinationZone != model.ZoneGraveyard || got.DestinationFace != model.CardFaceUp {
		t.Fatalf("MoveCard = %#v", got)
	}
}

func TestDragGhostClearsOnCancel(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "hand-1", CardID: "visible-hand-card", Owner: "player-one", ShowFace: true},
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		MoveCard: func(model.MoveCardCommand, model.Revision) {},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()
	window.Resize(fyne.NewSize(1200, 900))

	tiles := collectCardTiles(screen.playerBoards[1].hand)
	if len(tiles) == 0 {
		t.Fatal("expected hand tiles")
	}
	screen.beginCardDrag(tiles[0], model.ZoneHand, match.Players[0].ID)
	if screen.drag.ghost == nil {
		t.Fatal("expected drag ghost")
	}
	screen.cancelCardDrag()
	if screen.drag.active || screen.drag.ghost != nil || len(screen.dragLayer.Objects) != 0 {
		t.Fatal("expected cancel to clear ghost and drag session")
	}
}
