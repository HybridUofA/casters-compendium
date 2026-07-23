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

// uniqueTestCardIDs returns stable distinct identifiers for capacity tests.
func uniqueTestCardIDs(count int) []string {
	cardIDs := make([]string, count)
	for index := range cardIDs {
		cardIDs[index] = fmt.Sprintf("card-%02d", index)
	}
	return cardIDs
}
