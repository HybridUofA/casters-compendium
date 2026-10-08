package ui

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestEditableAetherPoolShowsAllNineTypes(t *testing.T) {
	display := newAetherPoolDisplay(model.AetherPool{}, true, func(string, int) {})
	box, ok := display.(*fyne.Container)
	if !ok {
		t.Fatalf("display type = %T; want *fyne.Container", display)
	}
	if len(box.Objects) != 10 {
		t.Fatalf("editable pool children = %d; want 10 (label + 9 types)", len(box.Objects))
	}
	interactive := 0
	for _, object := range box.Objects[1:] {
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
	if !ok {
		t.Fatalf("display type = %T; want *fyne.Container", display)
	}
	if len(box.Objects) != 2 {
		t.Fatalf("view-only pool children = %d; want 2 (label + Aes)", len(box.Objects))
	}
}
