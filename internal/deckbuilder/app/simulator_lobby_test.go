package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestRoomMatchesSearchByNameCreatorOrCode(t *testing.T) {
	room := protocol.RoomSummary{
		RoomCode: "ABCD",
		RoomName: "Friday Night Casters",
		HostName: "Hybrid",
	}
	tests := []struct {
		query string
		want  bool
	}{
		{query: "friday", want: true},
		{query: "hybrid", want: true},
		{query: "abcd", want: true},
		{query: "missing", want: false},
	}
	for _, testCase := range tests {
		if got := roomMatchesSearch(room, testCase.query); got != testCase.want {
			t.Fatalf("roomMatchesSearch(%q) = %v; want %v", testCase.query, got, testCase.want)
		}
	}
}

func TestShouldAutoPassPriorityForViewDefaultsToEmptyChase(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, "") // force default

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseDraw,
		},
	}
	if !shouldAutoPassPriorityForView(base) {
		t.Fatal("default empty mode should auto-pass Draw with an empty Chase")
	}

	withChase := base
	withChase.ChaseLinkCount = 1
	if shouldAutoPassPriorityForView(withChase) {
		t.Fatal("default empty mode must not auto-pass after a Chase starts")
	}

	mainTurn := base
	mainTurn.Turn.Phase = model.PhaseMain
	if shouldAutoPassPriorityForView(mainTurn) {
		t.Fatal("default empty mode must keep Main priority for the turn player")
	}

	mainOpponent := mainTurn
	mainOpponent.Turn.ActivePlayer = "player-two"
	if !shouldAutoPassPriorityForView(mainOpponent) {
		t.Fatal("default empty mode should auto-pass Main for the non-turn player")
	}
}

func TestShouldAutoPassPriorityForViewHonorsFullAndStopsModes(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()

	match := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseDraw,
		},
	}

	prefs.SetString(networkPriorityModePreferenceKey, priorityModeFull)
	if shouldAutoPassPriorityForView(match) {
		t.Fatal("full control must never auto-pass")
	}

	prefs.SetString(networkPriorityModePreferenceKey, priorityModeStops)
	prefs.SetString(networkStopPhasesPreferenceKey, "Main,Battle")
	if !shouldAutoPassPriorityForView(match) {
		t.Fatal("stops mode should auto-pass Draw")
	}
	match.Turn.Phase = model.PhaseMain
	if shouldAutoPassPriorityForView(match) {
		t.Fatal("stops mode must not auto-pass configured Main stop")
	}
}
