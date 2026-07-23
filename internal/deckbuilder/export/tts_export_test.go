package deckexport

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestGenDeckIDsEmpty verifies an empty zone produces no physical cards or sheet faces.
func TestGenDeckIDsEmpty(t *testing.T) {
	deckIDs, sheetIDs, err := genDeckIDs(nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(deckIDs) != 0 || len(sheetIDs) != 0 {
		t.Fatalf("genDeckIDs() = %v, %v; want empty results", deckIDs, sheetIDs)
	}
}

// TestGenDeckIDsUniqueCards verifies unique faces receive consecutive slots in physical order.
func TestGenDeckIDsUniqueCards(t *testing.T) {
	deckIDs, sheetIDs, err := genDeckIDs(
		[]string{"abolition", "pentachi", "grace"},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}

	wantDeckIDs := []int{100, 101, 102}
	if !slices.Equal(deckIDs, wantDeckIDs) {
		t.Fatalf("physical deck IDs = %v, want %v", deckIDs, wantDeckIDs)
	}
	wantSheetIDs := []string{"abolition", "pentachi", "grace"}
	if !slices.Equal(sheetIDs, wantSheetIDs) {
		t.Fatalf("unique sheet IDs = %v, want %v", sheetIDs, wantSheetIDs)
	}
}

// TestGenDeckIDsDuplicates verifies repeated physical cards reuse their first face-sheet slot.
func TestGenDeckIDsDuplicates(t *testing.T) {
	deckIDs, sheetIDs, err := genDeckIDs(
		[]string{"abolition", "pentachi", "pentachi", "abolition", "grace"},
		1,
	)
	if err != nil {
		t.Fatal(err)
	}

	wantDeckIDs := []int{100, 101, 101, 100, 102}
	if !slices.Equal(deckIDs, wantDeckIDs) {
		t.Fatalf("physical deck IDs = %v, want %v", deckIDs, wantDeckIDs)
	}
	wantSheetIDs := []string{"abolition", "pentachi", "grace"}
	if !slices.Equal(sheetIDs, wantSheetIDs) {
		t.Fatalf("unique sheet IDs = %v, want %v", sheetIDs, wantSheetIDs)
	}
}

// TestGenDeckIDsUsesDeckKey verifies zones can use distinct TTS custom-deck namespaces.
func TestGenDeckIDsUsesDeckKey(t *testing.T) {
	deckIDs, sheetIDs, err := genDeckIDs([]string{"call-forth", "call-forth"}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(deckIDs, []int{200, 200}) {
		t.Fatalf("physical deck IDs = %v, want [200 200]", deckIDs)
	}
	if !slices.Equal(sheetIDs, []string{"call-forth"}) {
		t.Fatalf("unique sheet IDs = %v, want [call-forth]", sheetIDs)
	}
}

// TestGenDeckIDsAllowsFullSheet verifies every slot from zero through the capacity limit is usable.
func TestGenDeckIDsAllowsFullSheet(t *testing.T) {
	cardIDs := uniqueTestCardIDs(ttsSheetMaxCards)

	deckIDs, sheetIDs, err := genDeckIDs(cardIDs, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(deckIDs) != ttsSheetMaxCards || len(sheetIDs) != ttsSheetMaxCards {
		t.Fatalf(
			"result lengths = %d physical, %d unique; want %d each",
			len(deckIDs),
			len(sheetIDs),
			ttsSheetMaxCards,
		)
	}
	if deckIDs[0] != 100 || deckIDs[len(deckIDs)-1] != 100+ttsSheetMaxCards-1 {
		t.Fatalf("physical deck ID bounds = %d..%d", deckIDs[0], deckIDs[len(deckIDs)-1])
	}
	if !slices.Equal(sheetIDs, cardIDs) {
		t.Fatal("full sheet did not preserve unique card order")
	}
}

// TestGenDeckIDsAllowsDuplicatesAfterFullSheet verifies duplicates do not consume new slots.
func TestGenDeckIDsAllowsDuplicatesAfterFullSheet(t *testing.T) {
	cardIDs := append(uniqueTestCardIDs(ttsSheetMaxCards), "card-00")

	deckIDs, sheetIDs, err := genDeckIDs(cardIDs, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sheetIDs) != ttsSheetMaxCards {
		t.Fatalf("unique sheet length = %d, want %d", len(sheetIDs), ttsSheetMaxCards)
	}
	if got := deckIDs[len(deckIDs)-1]; got != 100 {
		t.Fatalf("duplicate physical ID = %d, want 100", got)
	}
}

// TestGenDeckIDsRejectsSheetOverflow verifies the first face beyond capacity is rejected.
func TestGenDeckIDsRejectsSheetOverflow(t *testing.T) {
	cardIDs := uniqueTestCardIDs(ttsSheetMaxCards + 1)

	deckIDs, sheetIDs, err := genDeckIDs(cardIDs, 1)
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("max %d", ttsSheetMaxCards)) {
		t.Fatalf("genDeckIDs() error = %v", err)
	}
	if deckIDs != nil || sheetIDs != nil {
		t.Fatalf("error results = %v, %v; want nil results", deckIDs, sheetIDs)
	}
}

// TestGenDeckIDsRejectsInvalidDeckKey verifies zero and negative namespaces are invalid.
func TestGenDeckIDsRejectsInvalidDeckKey(t *testing.T) {
	for _, deckKey := range []int{0, -1} {
		t.Run(fmt.Sprintf("key_%d", deckKey), func(t *testing.T) {
			deckIDs, sheetIDs, err := genDeckIDs([]string{"abolition"}, deckKey)
			if err == nil || !strings.Contains(err.Error(), "deck key must be positive") {
				t.Fatalf("genDeckIDs() error = %v", err)
			}
			if deckIDs != nil || sheetIDs != nil {
				t.Fatalf("error results = %v, %v; want nil results", deckIDs, sheetIDs)
			}
		})
	}
}

// TestGenDeckIDsRejectsBlankCardID verifies malformed input fails atomically.
func TestGenDeckIDsRejectsBlankCardID(t *testing.T) {
	deckIDs, sheetIDs, err := genDeckIDs(
		[]string{"abolition", "", "pentachi"},
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "cannot be blank") {
		t.Fatalf("genDeckIDs() error = %v", err)
	}
	if deckIDs != nil || sheetIDs != nil {
		t.Fatalf("error results = %v, %v; want nil results", deckIDs, sheetIDs)
	}
}

// TestSheetDimensions verifies compact single-row sheets and ten-column
// multi-row sheets use ceiling division at every important row boundary.
func TestSheetDimensions(t *testing.T) {
	tests := []struct {
		cardCount  int
		wantWidth  int
		wantHeight int
	}{
		{cardCount: 1, wantWidth: 1, wantHeight: 1},
		{cardCount: 2, wantWidth: 2, wantHeight: 1},
		{cardCount: 9, wantWidth: 9, wantHeight: 1},
		{cardCount: 10, wantWidth: 10, wantHeight: 1},
		{cardCount: 11, wantWidth: 10, wantHeight: 2},
		{cardCount: 19, wantWidth: 10, wantHeight: 2},
		{cardCount: 20, wantWidth: 10, wantHeight: 2},
		{cardCount: 21, wantWidth: 10, wantHeight: 3},
		{cardCount: 50, wantWidth: 10, wantHeight: 5},
		{cardCount: 69, wantWidth: 10, wantHeight: 7},
		{cardCount: 70, wantWidth: 10, wantHeight: 7},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d_cards", test.cardCount), func(t *testing.T) {
			width, height, err := sheetDimensions(test.cardCount)
			if err != nil {
				t.Fatal(err)
			}
			if width != test.wantWidth || height != test.wantHeight {
				t.Fatalf(
					"sheetDimensions(%d) = %dx%d, want %dx%d",
					test.cardCount,
					width,
					height,
					test.wantWidth,
					test.wantHeight,
				)
			}
			if width*height < test.cardCount {
				t.Fatalf(
					"sheetDimensions(%d) capacity = %d, too small",
					test.cardCount,
					width*height,
				)
			}
		})
	}
}

// TestSheetDimensionsRejectsInvalidCount verifies empty, negative, and
// over-capacity sheets fail without returning usable dimensions.
func TestSheetDimensionsRejectsInvalidCount(t *testing.T) {
	tests := []struct {
		name      string
		cardCount int
		wantError string
	}{
		{name: "zero", cardCount: 0, wantError: "must be positive"},
		{name: "negative", cardCount: -1, wantError: "must be positive"},
		{name: "over capacity", cardCount: ttsSheetMaxCards + 1, wantError: fmt.Sprintf("%d", ttsSheetMaxCards)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			width, height, err := sheetDimensions(test.cardCount)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("sheetDimensions(%d) error = %v", test.cardCount, err)
			}
			if width != 0 || height != 0 {
				t.Fatalf(
					"sheetDimensions(%d) = %dx%d after error, want 0x0",
					test.cardCount,
					width,
					height,
				)
			}
		})
	}
}

// TestBuildCustomDeckState verifies TTS sheet metadata is populated from the
// selected asset paths and calculated grid dimensions.
func TestBuildCustomDeckState(t *testing.T) {
	state, err := buildCustomDeckState(
		"/tmp/Caster Images/main deck.png",
		"/tmp/Caster Images/card back.png",
		11,
	)
	if err != nil {
		t.Fatal(err)
	}

	if state.FaceURL != "/tmp/Caster Images/main deck.png" {
		t.Fatalf("FaceURL = %q", state.FaceURL)
	}
	if state.BackURL != "/tmp/Caster Images/card back.png" {
		t.Fatalf("BackURL = %q", state.BackURL)
	}
	if state.NumWidth != 10 || state.NumHeight != 2 {
		t.Fatalf("dimensions = %dx%d, want 10x2", state.NumWidth, state.NumHeight)
	}
	if !state.BackIsHidden {
		t.Fatal("BackIsHidden = false, want true")
	}
	if state.UniqueBack {
		t.Fatal("UniqueBack = true, want false")
	}
	if state.Type != 0 {
		t.Fatalf("Type = %d, want 0", state.Type)
	}
}

// TestBuildCustomDeckStateUsesCompactSingleRow verifies small sheets do not
// retain unused columns from the ten-column maximum.
func TestBuildCustomDeckStateUsesCompactSingleRow(t *testing.T) {
	state, err := buildCustomDeckState("faces.png", "back.png", 5)
	if err != nil {
		t.Fatal(err)
	}
	if state.NumWidth != 5 || state.NumHeight != 1 {
		t.Fatalf("dimensions = %dx%d, want 5x1", state.NumWidth, state.NumHeight)
	}
}

// TestBuildCustomDeckStateAllowsMaximumSheet verifies the 70th face remains valid.
func TestBuildCustomDeckStateAllowsMaximumSheet(t *testing.T) {
	state, err := buildCustomDeckState(
		"faces.png",
		"back.png",
		ttsSheetMaxCards,
	)
	if err != nil {
		t.Fatal(err)
	}
	if state.NumWidth != ttsSheetColumns || state.NumHeight != ttsSheetMaxRows {
		t.Fatalf(
			"dimensions = %dx%d, want %dx%d",
			state.NumWidth,
			state.NumHeight,
			ttsSheetColumns,
			ttsSheetMaxRows,
		)
	}
}

// TestBuildCustomDeckStateRejectsMissingAssetPath verifies neither required
// image reference can be omitted.
func TestBuildCustomDeckStateRejectsMissingAssetPath(t *testing.T) {
	tests := []struct {
		name      string
		faceURL   string
		backURL   string
		wantError string
	}{
		{name: "missing face", faceURL: "", backURL: "back.png", wantError: "face path"},
		{name: "missing back", faceURL: "faces.png", backURL: "", wantError: "back path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, err := buildCustomDeckState(
				test.faceURL,
				test.backURL,
				1,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("buildCustomDeckState() error = %v", err)
			}
			if state != (CustomDeckState{}) {
				t.Fatalf("error state = %#v, want zero value", state)
			}
		})
	}
}

// TestBuildCustomDeckStateRejectsInvalidCount verifies dimension errors are
// propagated without returning partially populated TTS state.
func TestBuildCustomDeckStateRejectsInvalidCount(t *testing.T) {
	for _, cardCount := range []int{0, -1, ttsSheetMaxCards + 1} {
		t.Run(fmt.Sprintf("%d_cards", cardCount), func(t *testing.T) {
			state, err := buildCustomDeckState("faces.png", "back.png", cardCount)
			if err == nil {
				t.Fatalf("buildCustomDeckState(%d) error = nil", cardCount)
			}
			if state != (CustomDeckState{}) {
				t.Fatalf("error state = %#v, want zero value", state)
			}
		})
	}
}

// uniqueTestCardIDs returns stable distinct identifiers for capacity tests.
func uniqueTestCardIDs(count int) []string {
	cardIDs := make([]string, count)
	for index := range cardIDs {
		cardIDs[index] = fmt.Sprintf("card-%02d", index)
	}
	return cardIDs
}
