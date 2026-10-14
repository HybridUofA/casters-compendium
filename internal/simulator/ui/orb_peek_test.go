package ui

import (
	"testing"

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

	opponentBoard.beginPeekOrbPrompt()
	if screen.orbPickMode != orbPickPeek {
		t.Fatalf("orbPickMode = %d; want peek", screen.orbPickMode)
	}
	// Click the second Orb tile to finish the pick (no popup).
	screen.handleOpponentOrbPrimaryTap(1)
	if peekedOwner != opponentBoard.player.ID || peekedIndex != 1 {
		t.Fatalf("peek owner/index = %s/%d; want %s/1", peekedOwner, peekedIndex, opponentBoard.player.ID)
	}
	if screen.orbPickMode != orbPickNone {
		t.Fatal("orb pick mode should clear after peek")
	}
}

func TestCorruptOrbClickSubmitsSelectedOrbIndex(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 55
	match.Turn = model.TurnState{Number: 2, ActivePlayer: match.ViewerID, Phase: model.PhaseBattle}
	match.Attack = model.AttackState{
		AttackerID:   "p1-attacker",
		TargetKind:   model.AttackTargetPlayer,
		Step:         model.BattleStepAwaitingJudgment,
		CorruptCount: 1,
	}
	match.PrioritySequenceOpen = false
	match.Players[1].Orbs = []simulatorview.CardView{
		{Face: model.CardFaceDown},
		{Face: model.CardFaceDown},
		{Face: model.CardFaceDown},
	}
	var (
		orbIndexes []int
		revision   model.Revision
	)
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CorruptOrbs: func(gotIndexes []int, gotRevision model.Revision) {
			orbIndexes = append([]int(nil), gotIndexes...)
			revision = gotRevision
		}},
		nil,
	)
	window := test.NewWindow(controller.Content())
	defer window.Close()

	controller.handleOpponentOrbPrimaryTap(1)
	if len(orbIndexes) != 1 || orbIndexes[0] != 1 || revision != 55 {
		t.Fatalf("CorruptOrbs submitted %#v/%d; want [1]/55", orbIndexes, revision)
	}
}
