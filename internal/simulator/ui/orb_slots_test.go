package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestReconcileOrbSlotIDsKeepsStableHoles(t *testing.T) {
	initial := []simulatorview.CardView{
		{MatchID: "a"}, {MatchID: "b"}, {MatchID: "c"},
		{MatchID: "d"}, {MatchID: "e"}, {MatchID: "f"}, {MatchID: "g"},
	}
	slots := reconcileOrbSlotIDs(nil, initial)
	want := []model.MatchCardID{"a", "b", "c", "d", "e", "f", "g"}
	for index, id := range want {
		if slots[index] != id {
			t.Fatalf("initial slot %d = %q; want %q", index, slots[index], id)
		}
	}

	// Remove middle orbs b and e — their slots must stay empty.
	remaining := []simulatorview.CardView{
		{MatchID: "a"}, {MatchID: "c"}, {MatchID: "d"}, {MatchID: "f"}, {MatchID: "g"},
	}
	slots = reconcileOrbSlotIDs(slots, remaining)
	want = []model.MatchCardID{"a", "", "c", "d", "", "f", "g"}
	for index, id := range want {
		if slots[index] != id {
			t.Fatalf("after removals slot %d = %q; want %q", index, slots[index], id)
		}
	}

	views := orbViewsForSlots(slots, remaining)
	if views[1].MatchID != "" || views[4].MatchID != "" {
		t.Fatal("empty slots must project empty CardViews")
	}
	if views[2].MatchID != "c" || views[5].MatchID != "f" {
		t.Fatalf("remaining orbs shifted; got %#v", views)
	}
}

func TestOrbZoneKeepsPlaceholdersAfterRemoval(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[1].Orbs = []simulatorview.CardView{
		{MatchID: "p2-orb-0", Face: model.CardFaceDown},
		{MatchID: "p2-orb-1", Face: model.CardFaceDown},
		{MatchID: "p2-orb-2", Face: model.CardFaceDown},
		{MatchID: "p2-orb-3", Face: model.CardFaceDown},
		{MatchID: "p2-orb-4", Face: model.CardFaceDown},
		{MatchID: "p2-orb-5", Face: model.CardFaceDown},
		{MatchID: "p2-orb-6", Face: model.CardFaceDown},
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	opponent := screen.playerBoards[0]
	stack := findVerticalCardStack(opponent.orbs)
	if stack == nil {
		t.Fatal("expected layered orb stack")
	}
	if len(stack.Objects) != orbSlotCount {
		t.Fatalf("orb objects = %d; want %d prize slots", len(stack.Objects), orbSlotCount)
	}
	firstTiles := collectCardTiles(stack)
	if len(firstTiles) != 7 {
		t.Fatalf("orb tiles = %d; want 7", len(firstTiles))
	}
	positions := make([]fyne.Position, len(firstTiles))
	for index, tile := range firstTiles {
		positions[index] = tile.Position()
	}

	updated := match
	updated.Players[1].Orbs = []simulatorview.CardView{
		{MatchID: "p2-orb-0", Face: model.CardFaceDown},
		{MatchID: "p2-orb-2", Face: model.CardFaceDown},
		{MatchID: "p2-orb-3", Face: model.CardFaceDown},
		{MatchID: "p2-orb-4", Face: model.CardFaceDown},
		{MatchID: "p2-orb-5", Face: model.CardFaceDown},
		{MatchID: "p2-orb-6", Face: model.CardFaceDown},
	}
	screen.Update(updated)

	stack = findVerticalCardStack(screen.playerBoards[0].orbs)
	if stack == nil {
		t.Fatal("expected layered orb stack after update")
	}
	placeholders := 0
	for _, object := range stack.Objects {
		if _, ok := object.(*orbSlotPlaceholder); ok {
			placeholders++
		}
	}
	if placeholders != 1 {
		t.Fatalf("placeholders = %d; want 1 empty prize slot", placeholders)
	}
	tiles := collectCardTiles(stack)
	if len(tiles) != 6 {
		t.Fatalf("orb tiles = %d; want 6 remaining", len(tiles))
	}
	// Slot 0 (orb-0) and slot 2 (orb-2) must keep their Y positions from before.
	var orb0, orb2 *CardTile
	for _, tile := range tiles {
		switch tile.View.MatchID {
		case "p2-orb-0":
			orb0 = tile
		case "p2-orb-2":
			orb2 = tile
		}
	}
	if orb0 == nil || orb2 == nil {
		t.Fatal("expected orb-0 and orb-2 to remain")
	}
	if orb0.Position().Y != positions[0].Y {
		t.Fatalf("orb-0 moved from Y %.1f to %.1f", positions[0].Y, orb0.Position().Y)
	}
	if orb2.Position().Y != positions[2].Y {
		t.Fatalf("orb-2 moved from Y %.1f to %.1f; hole should preserve prize slot", positions[2].Y, orb2.Position().Y)
	}
}

func TestOrbViewsForSlotsPacksAnonymousFaceDownOrbs(t *testing.T) {
	// Pre-1.2.1 hosts projected unknown Orbs with empty MatchID.
	anonymous := []simulatorview.CardView{
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
	}
	slots := reconcileOrbSlotIDs(nil, anonymous)
	for _, id := range slots {
		if id != "" {
			t.Fatalf("anonymous orbs must not claim prize MatchIDs; got %q", id)
		}
	}
	views := orbViewsForSlots(slots, anonymous)
	occupied := 0
	for _, view := range views {
		if !orbSlotOccupied(view) {
			continue
		}
		occupied++
		if view.MatchID != "" || view.Face != model.CardFaceDown || view.ShowFace {
			t.Fatalf("anonymous packed orb = %#v; want face-down card back", view)
		}
	}
	if occupied != 7 {
		t.Fatalf("occupied anonymous slots = %d; want 7 card backs", occupied)
	}
}

func TestOrbZoneRendersCardBacksForAnonymousMatchIDs(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	// Simulate a legacy host that omits Orb MatchIDs the viewer does not know.
	match.Players[1].Orbs = []simulatorview.CardView{
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
		{Face: model.CardFaceDown, ShowFace: false},
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	stack := findVerticalCardStack(screen.playerBoards[0].orbs)
	if stack == nil {
		t.Fatal("expected layered orb stack")
	}
	placeholders := 0
	for _, object := range stack.Objects {
		if _, ok := object.(*orbSlotPlaceholder); ok {
			placeholders++
		}
	}
	if placeholders != 0 {
		t.Fatalf("placeholders = %d; want 0 (anonymous orbs should show card backs)", placeholders)
	}
	tiles := collectCardTiles(stack)
	if len(tiles) != 7 {
		t.Fatalf("orb tiles = %d; want 7 card-back tiles", len(tiles))
	}
	for index, tile := range tiles {
		if tile.View.ShowFace {
			t.Fatalf("orb %d ShowFace = true; want card back", index)
		}
		if tile.sidewaysImage == nil && (tile.image == nil || (tile.image.Image == nil && tile.image.Resource == nil)) {
			t.Fatalf("orb %d has no card-back artwork", index)
		}
	}
}

func TestOrbTargetHoverHighlightsOnMouseIn(t *testing.T) {
	tile := NewCardTile(
		simulatorview.CardView{MatchID: "orb", Face: model.CardFaceDown},
		cards.Card{},
		fyne.NewSize(orbCardWidth, orbCardHeight),
		nil,
		nil,
	)
	tile.SetTargetHoverEnabled(true)
	if tile.selectionBorder.Visible() {
		t.Fatal("outline should stay hidden until hover")
	}
	tile.MouseIn(nil)
	if !tile.selectionBorder.Visible() {
		t.Fatal("targetable orb should outline on hover")
	}
	tile.MouseOut()
	if tile.selectionBorder.Visible() {
		t.Fatal("outline should clear on mouse out")
	}
	tile.SetSelected(true)
	tile.MouseOut()
	if !tile.selectionBorder.Visible() {
		t.Fatal("selected outline should survive mouse out")
	}
}
