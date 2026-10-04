package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestOpponentOrbZoneOffersPeekWhenWired(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[1].Orbs = []simulatorview.CardView{
		{Face: model.CardFaceDown},
		{Face: model.CardFaceDown},
	}
	var peekedOwner model.PlayerID
	var peekedIndex int = -1
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			PeekOrb: func(
				ownerID model.PlayerID,
				orbIndex int,
				_ model.Revision,
				done func(simulatorview.CardView, error),
			) {
				peekedOwner = ownerID
				peekedIndex = orbIndex
				if done != nil {
					done(simulatorview.CardView{
						MatchID:  "p2-orb-b",
						CardID:   "definition",
						ShowFace: true,
					}, nil)
				}
			},
		},
		nil,
	)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	opponentBoard := screen.playerBoards[0]
	if findZoneInteractLayer(opponentBoard.orbs) == nil {
		t.Fatal("opponent orb zone should offer peek context actions")
	}

	originalPrompt := promptOrbIndex
	promptOrbIndex = func(window fyne.Window, prompt string, orbCount int, onConfirm func(int)) {
		if orbCount != 2 {
			t.Fatalf("orbCount = %d; want 2", orbCount)
		}
		onConfirm(1)
	}
	t.Cleanup(func() { promptOrbIndex = originalPrompt })

	opponentBoard.beginPeekOrbPrompt()
	if peekedOwner != opponentBoard.player.ID || peekedIndex != 1 {
		t.Fatalf("peek owner/index = %s/%d; want %s/1", peekedOwner, peekedIndex, opponentBoard.player.ID)
	}
}
