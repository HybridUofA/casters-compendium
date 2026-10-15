package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestEditableAetherPoolShowsAllNineTypes(t *testing.T) {
	display := newAetherPoolDisplay(model.AetherPool{}, true, func(string, int) {})
	box, ok := display.(*fyne.Container)
	if !ok || len(box.Objects) != 2 {
		t.Fatalf("display = %#v; want VBox(label, grid)", display)
	}
	grid, ok := box.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("grid type = %T; want *fyne.Container", box.Objects[1])
	}
	if len(grid.Objects) != 9 {
		t.Fatalf("editable pool chips = %d; want 9 types", len(grid.Objects))
	}
	interactive := 0
	for _, object := range grid.Objects {
		if _, ok := object.(*zoneInteractLayer); ok {
			interactive++
		}
	}
	if interactive != 9 {
		t.Fatalf("interactive aether icons = %d; want 9", interactive)
	}
}

func TestViewOnlyAetherPoolHidesZeros(t *testing.T) {
	display := newAetherPoolDisplay(model.AetherPool{Aes: 2}, false, nil)
	box, ok := display.(*fyne.Container)
	if !ok || len(box.Objects) != 2 {
		t.Fatalf("display = %#v; want VBox(label, grid)", display)
	}
	grid, ok := box.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("grid type = %T; want *fyne.Container", box.Objects[1])
	}
	if len(grid.Objects) != 1 {
		t.Fatalf("view-only pool chips = %d; want 1 (Aes)", len(grid.Objects))
	}
}

func TestAetherPoolsDoNotUseHorizontalScroll(t *testing.T) {
	match := testMatchView()
	match.Players[0].Aether = model.AetherPool{
		Aes: 1, Aqua: 1, Ignus: 1, Luna: 1, Silva: 1,
		Solis: 1, Terra: 1, Void: 1, NonElemental: 1,
	}
	screen := NewBoardController(match, testDefinitions(), BoardActions{
		AdjustAether: func(model.AdjustAetherCommand, model.Revision) {},
	}, nil)
	window := test.NewWindow(screen.Content())
	defer window.Close()
	window.Resize(fyne.NewSize(1200, 900))

	if scroll := findScroll(screen.aetherPools); scroll != nil {
		t.Fatal("Aether pools should not use a scrollbar")
	}

	display := newAetherPoolDisplay(match.Players[0].Aether, true, func(string, int) {})
	box := display.(*fyne.Container)
	grid := box.Objects[1].(*fyne.Container)
	grid.Resize(fyne.NewSize(previewPanelWidth-24, 120))
	rows := map[float32]int{}
	for _, object := range grid.Objects {
		rows[object.Position().Y]++
	}
	if len(rows) != 2 {
		t.Fatalf("aether chip rows = %d; want 5 columns × 2 rows for 9 types", len(rows))
	}
}
