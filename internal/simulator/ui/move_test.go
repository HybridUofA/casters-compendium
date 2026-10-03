package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestManualMovePanelSubmitsPlacementAndResetsOnRevision(t *testing.T) {
	match := simulatorview.MatchView{ViewerID: "one", MatchStatus: model.StatusInProgress, Revision: 4, Players: [2]simulatorview.PlayerView{
		{ID: "one", OpeningHandFinalized: true, ServantZone: []simulatorview.CardView{{MatchID: "servant", CardID: "definition", Face: model.CardFaceUp, Orientation: model.OrientationRested, ShowFace: true}}},
		{ID: "two", OpeningHandFinalized: true, Hand: []simulatorview.CardView{{}}},
	}}
	var received model.MoveCardCommand
	var revision model.Revision
	screen := NewBoardController(match, []gamecards.Card{{ID: "definition", Name: "Example", Type: "Servant"}}, BoardActions{MoveCard: func(c model.MoveCardCommand, r model.Revision) { received = c; revision = r }}, nil)
	button := findButton(screen.Content(), "Move a card…")
	if button == nil {
		t.Fatal("missing manual move action")
	}
	test.Tap(button)
	choices := findSelects(screen.preview.manualActions)
	if len(choices) != 6 || len(choices[0].Options) != 1 {
		t.Fatal("unexpected choices or hidden opponent cards exposed")
	}
	confirm := findButton(screen.preview.manualActions, "Confirm move")
	if !confirm.Disabled() {
		t.Fatal("incomplete move enabled")
	}
	choices[0].SetSelected(choices[0].Options[0])
	choices[1].SetSelected(string(model.ZoneDeck))
	choices[5].SetSelected(string(model.DeckPlacementBottom))
	if confirm.Disabled() {
		t.Fatal("valid deck selection disabled")
	}
	test.Tap(confirm)
	if received.CardID != "servant" || received.Placement != model.DeckPlacementBottom || received.DestinationFace != model.CardFaceDown || received.EntryOrientation != "" || revision != 4 {
		t.Fatalf("command=%+v revision=%d", received, revision)
	}
	choices[1].SetSelected(string(model.ZoneServant))
	choices[2].SetSelected("Opponent's side")
	if !choices[4].Disabled() || choices[4].Selected != string(model.OrientationRested) {
		t.Fatal("transfer did not preserve orientation")
	}
	test.Tap(confirm)
	if received.DestinationPlayerID != "two" || received.EntryOrientation != model.OrientationRested || received.Placement != "" {
		t.Fatalf("transfer=%+v", received)
	}
	choices[1].SetSelected(string(model.ZoneCaster))
	choices[2].SetSelected("Your side")
	test.Tap(confirm)
	if received.DestinationFace != model.CardFaceDown || received.DestinationZone != model.ZoneCaster {
		t.Fatalf("conversion=%+v", received)
	}
	match.Revision++
	screen.Update(match)
	if findButton(screen.preview.manualActions, "Confirm move") != nil {
		t.Fatal("stale move form survived update")
	}
	if minimum := screen.Content().MinSize(); minimum.Width > 1400 || minimum.Height > 850 {
		t.Fatalf("board no longer fits: %v", minimum)
	}
}

func TestManualMovePanelExplainsAndHidesUnsupportedChoices(t *testing.T) {
	match := simulatorview.MatchView{ViewerID: "one", MatchStatus: model.StatusInProgress, Players: [2]simulatorview.PlayerView{
		{
			ID:                   "one",
			OpeningHandFinalized: true,
			Hand:                 []simulatorview.CardView{{MatchID: "hand", CardID: "definition", Face: model.CardFaceDown}},
			DeckCount:            1,
			Orbs:                 []simulatorview.CardView{{Face: model.CardFaceDown}},
			Exile: []simulatorview.CardView{
				{MatchID: "visible-exile", CardID: "definition", Face: model.CardFaceUp, ShowFace: true},
				{Face: model.CardFaceDown},
			},
		},
		{ID: "two", OpeningHandFinalized: true},
	}}
	screen := NewBoardController(
		match,
		[]gamecards.Card{{ID: "definition", Name: "Example", Type: "Servant"}},
		BoardActions{MoveCard: func(model.MoveCardCommand, model.Revision) {}},
		nil,
	)

	test.Tap(findButton(screen.Content(), "Move a card…"))
	choices := findSelects(screen.preview.manualActions)
	if len(choices) != 6 {
		t.Fatalf("select count = %d; want 6", len(choices))
	}
	if len(choices[0].Options) != 2 {
		t.Fatalf("source count = %d; want visible Hand and Exile cards only", len(choices[0].Options))
	}
	for _, destination := range choices[1].Options {
		if destination == string(model.ZoneOrbs) {
			t.Fatal("unsupported Orb destination is selectable")
		}
	}
	for _, text := range []string{"Prototype limits", "Tokens", "Stock", "Orbs", "face-down Exile"} {
		if !containsTextPart(screen.preview.manualActions, text) {
			t.Fatalf("manual move panel does not explain %q limitation", text)
		}
	}
}
