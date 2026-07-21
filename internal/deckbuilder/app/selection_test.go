package deckbuilder

import (
	"slices"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// TestSelectedStateZeroValue verifies selection state is usable without a constructor.
func TestSelectedStateZeroValue(t *testing.T) {
	var selection SelectedState

	if got := selection.Count(); got != 0 {
		t.Fatalf("zero-value selection count = %d, want 0", got)
	}
	if selection.Contains(decks.MainZone, 0) {
		t.Fatal("zero-value selection unexpectedly contains main-deck index 0")
	}
	if got := selection.SortedIndices(); len(got) != 0 {
		t.Fatalf("zero-value sorted indices = %#v, want empty", got)
	}
	if selection.zone != decks.Zone("") {
		t.Fatalf("zero-value selection zone = %q, want empty", selection.zone)
	}
}

// TestSelectedStateToggle verifies additive selection, deselection, and final clearing.
func TestSelectedStateToggle(t *testing.T) {
	var selection SelectedState

	selection.Toggle(decks.MainZone, 4)
	assertSelectedState(t, &selection, decks.MainZone, []int{4})

	selection.Toggle(decks.MainZone, 1)
	assertSelectedState(t, &selection, decks.MainZone, []int{1, 4})

	selection.Toggle(decks.MainZone, 4)
	assertSelectedState(t, &selection, decks.MainZone, []int{1})
	if selection.Contains(decks.MainZone, 4) {
		t.Fatal("toggled-off main-deck index 4 remains selected")
	}

	selection.Toggle(decks.MainZone, 1)
	assertSelectedState(t, &selection, decks.Zone(""), nil)
}

// TestSelectedStateToggleChangesZone verifies a selection cannot span deck zones.
func TestSelectedStateToggleChangesZone(t *testing.T) {
	var selection SelectedState
	selection.Toggle(decks.MainZone, 1)
	selection.Toggle(decks.MainZone, 3)

	selection.Toggle(decks.SideZone, 2)

	assertSelectedState(t, &selection, decks.SideZone, []int{2})
	if selection.Contains(decks.MainZone, 1) || selection.Contains(decks.MainZone, 3) {
		t.Fatal("main-deck indices remained selected after switching to the side deck")
	}
}

// TestSelectedStateRejectsNegativeIndex verifies invalid UI positions do not mutate state.
func TestSelectedStateRejectsNegativeIndex(t *testing.T) {
	var selection SelectedState
	selection.Toggle(decks.MainZone, 2)

	selection.Toggle(decks.SideZone, -1)

	assertSelectedState(t, &selection, decks.MainZone, []int{2})
}

// TestSelectedStateClear verifies explicit clearing resets both membership and active zone.
func TestSelectedStateClear(t *testing.T) {
	var selection SelectedState
	selection.Toggle(decks.SideZone, 0)
	selection.Toggle(decks.SideZone, 5)

	selection.Clear()

	assertSelectedState(t, &selection, decks.Zone(""), nil)
	if selection.indices == nil {
		t.Fatal("Clear left the selected-index map nil")
	}
}

// TestSelectedStateSortedIndicesReturnsIndependentSlice verifies callers cannot mutate selection state.
func TestSelectedStateSortedIndicesReturnsIndependentSlice(t *testing.T) {
	var selection SelectedState
	selection.Toggle(decks.MainZone, 7)
	selection.Toggle(decks.MainZone, 1)
	selection.Toggle(decks.MainZone, 4)

	first := selection.SortedIndices()
	if !slices.Equal(first, []int{1, 4, 7}) {
		t.Fatalf("sorted indices = %#v, want [1 4 7]", first)
	}

	first[0] = 99
	second := selection.SortedIndices()
	if !slices.Equal(second, []int{1, 4, 7}) {
		t.Fatalf("selection changed through returned slice: %#v", second)
	}
}

// TestSelectedStateToggleNilReceiver verifies defensive nil handling does not panic.
func TestSelectedStateToggleNilReceiver(t *testing.T) {
	var selection *SelectedState
	selection.Toggle(decks.MainZone, 0)
}

func assertSelectedState(
	t *testing.T,
	selection *SelectedState,
	wantZone decks.Zone,
	wantIndices []int,
) {
	t.Helper()

	if selection.zone != wantZone {
		t.Fatalf("selection zone = %q, want %q", selection.zone, wantZone)
	}
	if got := selection.SortedIndices(); !slices.Equal(got, wantIndices) {
		t.Fatalf("selected indices = %#v, want %#v", got, wantIndices)
	}
	if got := selection.Count(); got != len(wantIndices) {
		t.Fatalf("selection count = %d, want %d", got, len(wantIndices))
	}
	for _, index := range wantIndices {
		if !selection.Contains(wantZone, index) {
			t.Fatalf("selection does not contain %s index %d", wantZone, index)
		}
	}
}
