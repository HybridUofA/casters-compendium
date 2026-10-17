package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestPendingChoiceSelectShowsCardNamesNotMatchIDs(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 90
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "player-1-card-26", CardID: "visible-hand-card", ShowFace: true},
		{MatchID: "player-1-card-27", CardID: "level-one-caster", ShowFace: true},
	}
	match.PendingChoice = model.PendingChoice{
		PlayerID: match.ViewerID,
		Kind:     "fx:discard_one_from_hand",
		Prompt:   "Discard a card",
		SelectableCardIDs: []model.MatchCardID{
			"player-1-card-26",
			"player-1-card-27",
		},
	}
	controller := NewBoardController(match, testDefinitions(), BoardActions{
		ResolvePendingChoice: func(string, []model.MatchCardID, model.Revision) {},
	}, nil)
	window := test.NewWindow(controller.Content())
	defer window.Close()

	selection := findSelectWithOptions(controller.Content(), []string{
		"Visible Hand Card",
		"Level One Caster",
	})
	if selection == nil {
		t.Fatal("pending choice select should list card names, not Match IDs")
	}
	for _, option := range selection.Options {
		if option == "player-1-card-26" || option == "player-1-card-27" {
			t.Fatalf("select still exposes Match ID %q", option)
		}
	}
}

func TestPendingChoiceHandClickDiscardsSelectedCard(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 91
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "player-1-card-26", CardID: "visible-hand-card", ShowFace: true},
		{MatchID: "player-1-card-27", CardID: "level-one-caster", ShowFace: true},
	}
	match.PendingChoice = model.PendingChoice{
		PlayerID: match.ViewerID,
		Kind:     "fx:discard_one_from_hand",
		Prompt:   "Discard a card",
		SelectableCardIDs: []model.MatchCardID{
			"player-1-card-26",
			"player-1-card-27",
		},
	}
	var (
		gotOption string
		gotCards  []model.MatchCardID
		gotRev    model.Revision
	)
	controller := NewBoardController(match, testDefinitions(), BoardActions{
		ResolvePendingChoice: func(optionID string, selected []model.MatchCardID, revision model.Revision) {
			gotOption = optionID
			gotCards = append([]model.MatchCardID(nil), selected...)
			gotRev = revision
		},
	}, nil)
	window := test.NewWindow(controller.Content())
	defer window.Close()

	tiles := collectCardTiles(controller.playerBoards[1].hand)
	if len(tiles) < 1 {
		t.Fatal("expected hand tiles")
	}
	var target *CardTile
	for _, tile := range tiles {
		if tile.View.MatchID == "player-1-card-26" {
			target = tile
			break
		}
	}
	if target == nil || target.OnActivate == nil {
		t.Fatal("expected clickable hand card for discard")
	}
	target.OnActivate()
	if gotOption != "" || gotRev != 91 || len(gotCards) != 1 || gotCards[0] != "player-1-card-26" {
		t.Fatalf("ResolvePendingChoice = option %q cards %#v rev %d", gotOption, gotCards, gotRev)
	}
}

func TestPendingChoiceSelectEntriesDedupesNames(t *testing.T) {
	screen := &BoardScreen{
		definitions: newCardLookup([]cards.Card{
			{ID: "a", Name: "Twin"},
			{ID: "b", Name: "Twin"},
		}),
		match: simulatorview.MatchView{
			Players: [2]simulatorview.PlayerView{
				{Hand: []simulatorview.CardView{
					{MatchID: "m1", CardID: "a", ShowFace: true},
					{MatchID: "m2", CardID: "b", ShowFace: true},
				}},
			},
		},
	}
	labels, byLabel := screen.pendingChoiceSelectEntries([]model.MatchCardID{"m1", "m2"})
	if len(labels) != 2 || labels[0] != "Twin" || labels[1] != "Twin (2)" {
		t.Fatalf("labels = %#v; want Twin / Twin (2)", labels)
	}
	if byLabel["Twin"] != "m1" || byLabel["Twin (2)"] != "m2" {
		t.Fatalf("byLabel = %#v", byLabel)
	}
}

func findSelectWithOptions(object fyne.CanvasObject, want []string) *widget.Select {
	wantSet := make(map[string]struct{}, len(want))
	for _, label := range want {
		wantSet[label] = struct{}{}
	}
	for _, selection := range findSelects(object) {
		if len(selection.Options) != len(want) {
			continue
		}
		matched := true
		for _, option := range selection.Options {
			if _, ok := wantSet[option]; !ok {
				matched = false
				break
			}
		}
		if matched {
			return selection
		}
	}
	return nil
}
