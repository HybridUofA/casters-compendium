package ui

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestMatchEndAnnouncementVictoryDefeatDraw(t *testing.T) {
	base := testMatchView()
	base.MatchStatus = model.StatusFinished
	base.Result = model.MatchResult{
		Winner: base.Players[0].ID,
		Loser:  base.Players[1].ID,
		Reason: model.EndReasonZeroOrbs,
	}

	title, body := matchEndAnnouncement(base)
	if title != "Victory!" || !strings.Contains(body, "defeated") {
		t.Fatalf("winner announcement = %q/%q", title, body)
	}

	base.ViewerID = base.Players[1].ID
	title, body = matchEndAnnouncement(base)
	if title != "Defeat" || !strings.Contains(body, "defeated you") {
		t.Fatalf("loser announcement = %q/%q", title, body)
	}

	base.Result = model.MatchResult{IsDraw: true, Reason: model.EndReasonSimultaneousLoss}
	title, body = matchEndAnnouncement(base)
	if title != "Draw" || !strings.Contains(body, "draw") {
		t.Fatalf("draw announcement = %q/%q", title, body)
	}
}

func TestBoardAnnouncesMatchResultOnce(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	finishedCalls := 0
	controller := NewBoardController(match, testDefinitions(), BoardActions{
		OnMatchFinished: func(simulatorview.MatchView) { finishedCalls++ },
	}, nil)

	finished := match
	finished.MatchStatus = model.StatusFinished
	finished.Revision++
	finished.Result = model.MatchResult{
		Winner: match.Players[0].ID,
		Loser:  match.Players[1].ID,
		Reason: model.EndReasonDeckOut,
	}
	controller.Update(finished)
	if !controller.resultAnnounced {
		t.Fatal("finished match did not mark result as announced")
	}
	controller.Update(finished)
	if finishedCalls != 0 {
		t.Fatalf("OnMatchFinished called before dialog confirm: %d", finishedCalls)
	}
}
