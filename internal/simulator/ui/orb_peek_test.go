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
		{MatchID: "p2-orb-a", Face: model.CardFaceDown},
		{MatchID: "p2-orb-b", Face: model.CardFaceDown},
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
	tile := findCardTileByMatchID(opponentBoard.orbs, "p2-orb-b")
	if tile == nil || tile.OnSecondaryActivate == nil {
		t.Fatal("opponent orb tile should offer right-click context actions")
	}
	screen.finishPeekOrb(1)
	if peekedOwner != opponentBoard.player.ID || peekedIndex != 1 {
		t.Fatalf("peek owner/index = %s/%d; want %s/1", peekedOwner, peekedIndex, opponentBoard.player.ID)
	}
}

func TestOrbRightClickCorruptAndBanishSubmitMatchIDs(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 9
	match.Players[0].Orbs = []simulatorview.CardView{
		{MatchID: "p1-orb-a", Face: model.CardFaceDown, Owner: match.Players[0].ID},
		{MatchID: "p1-orb-b", Face: model.CardFaceDown, Owner: match.Players[0].ID},
	}
	var corrupted, banished model.MatchCardID
	var revision model.Revision
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			ManualCorruptOrb: func(orbID model.MatchCardID, gotRevision model.Revision) {
				corrupted = orbID
				revision = gotRevision
			},
			ManualBanishOrb: func(orbID model.MatchCardID, gotRevision model.Revision) {
				banished = orbID
				revision = gotRevision
			},
		},
		nil,
	)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	viewerBoard := screen.playerBoards[1]
	tile := findCardTileByMatchID(viewerBoard.orbs, "p1-orb-b")
	if tile == nil || tile.OnSecondaryActivate == nil {
		t.Fatal("viewer orb tile should offer Corrupt/Banish")
	}
	event := fyne.PointEvent{}
	viewerBoard.showOrbTileContextMenu(1, tile.View, &event)
	screen.actions.ManualCorruptOrb("p1-orb-b", match.Revision)
	if corrupted != "p1-orb-b" || revision != 9 {
		t.Fatalf("corrupt = %q/%d; want p1-orb-b/9", corrupted, revision)
	}
	screen.actions.ManualBanishOrb("p1-orb-a", match.Revision)
	if banished != "p1-orb-a" {
		t.Fatalf("banish = %q; want p1-orb-a", banished)
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
		{MatchID: "p2-orb-0", Face: model.CardFaceDown},
		{MatchID: "p2-orb-1", Face: model.CardFaceDown},
		{MatchID: "p2-orb-2", Face: model.CardFaceDown},
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
		t.Fatalf("CorruptOrbs indexes/revision = %#v/%d; want [1]/55", orbIndexes, revision)
	}
}
