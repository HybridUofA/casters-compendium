package deckexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
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

// TestBuildCardObject verifies one physical TTS card references the expected
// custom-deck namespace and receives safe default transform values.
func TestBuildCardObject(t *testing.T) {
	state, err := buildCustomDeckState("main faces.png", "card back.png", 3)
	if err != nil {
		t.Fatal(err)
	}

	card, err := buildCardObject(102, "Grace", 1, state)
	if err != nil {
		t.Fatal(err)
	}
	if card.Name != "Card" {
		t.Fatalf("Name = %q, want Card", card.Name)
	}
	if card.Nickname != "Grace" {
		t.Fatalf("Nickname = %q, want Grace", card.Nickname)
	}
	if card.Description != "" {
		t.Fatalf("Description = %q, want empty", card.Description)
	}
	if card.CardID != 102 {
		t.Fatalf("CardID = %d, want 102", card.CardID)
	}
	if card.Transform.ScaleX != 1 ||
		card.Transform.ScaleY != 1 ||
		card.Transform.ScaleZ != 1 {
		t.Fatalf("Transform scale = %#v, want 1x1x1", card.Transform)
	}
	if len(card.CustomDeck) != 1 {
		t.Fatalf("CustomDeck length = %d, want 1", len(card.CustomDeck))
	}
	gotState, found := card.CustomDeck["1"]
	if !found {
		t.Fatalf("CustomDeck = %#v, want key 1", card.CustomDeck)
	}
	if gotState != state {
		t.Fatalf("CustomDeck[1] = %#v, want %#v", gotState, state)
	}
}

// TestBuildCardObjectUsesSuppliedNamespace verifies non-main deck keys are
// converted to decimal JSON map keys.
func TestBuildCardObjectUsesSuppliedNamespace(t *testing.T) {
	state, err := buildCustomDeckState("side faces.png", "card back.png", 12)
	if err != nil {
		t.Fatal(err)
	}

	card, err := buildCardObject(211, "Call Forth", 2, state)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := card.CustomDeck["2"]; !found {
		t.Fatalf("CustomDeck = %#v, want key 2", card.CustomDeck)
	}
}

// TestBuildCardObjectRejectsInvalidIdentity verifies IDs and custom-deck keys
// must be positive and must refer to the same hundreds namespace.
func TestBuildCardObjectRejectsInvalidIdentity(t *testing.T) {
	state, err := buildCustomDeckState("faces.png", "back.png", 1)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		ttsCardID int
		deckKey   int
		wantError string
	}{
		{name: "zero card ID", ttsCardID: 0, deckKey: 1, wantError: "card ID"},
		{name: "negative card ID", ttsCardID: -1, deckKey: 1, wantError: "card ID"},
		{name: "zero deck key", ttsCardID: 100, deckKey: 0, wantError: "deck key"},
		{name: "negative deck key", ttsCardID: 100, deckKey: -1, wantError: "deck key"},
		{name: "namespace mismatch", ttsCardID: 200, deckKey: 1, wantError: "does not match"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card, err := buildCardObject(
				test.ttsCardID,
				"Grace",
				test.deckKey,
				state,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("buildCardObject() error = %v", err)
			}
			if !reflect.DeepEqual(card, CardObject{}) {
				t.Fatalf("error card = %#v, want zero value", card)
			}
		})
	}
}

// TestBuildCardObjectRejectsMissingMetadata verifies a physical card cannot be
// emitted without a display name and both required TTS asset references.
func TestBuildCardObjectRejectsMissingMetadata(t *testing.T) {
	validState, err := buildCustomDeckState("faces.png", "back.png", 1)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		cardName  string
		state     CustomDeckState
		wantError string
	}{
		{name: "missing name", cardName: "", state: validState, wantError: "card name"},
		{
			name:     "missing face",
			cardName: "Grace",
			state: CustomDeckState{
				BackURL: "back.png",
			},
			wantError: "face path",
		},
		{
			name:     "missing back",
			cardName: "Grace",
			state: CustomDeckState{
				FaceURL: "faces.png",
			},
			wantError: "back path",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card, err := buildCardObject(100, test.cardName, 1, test.state)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("buildCardObject() error = %v", err)
			}
			if !reflect.DeepEqual(card, CardObject{}) {
				t.Fatalf("error card = %#v, want zero value", card)
			}
		})
	}
}

// TestBuildCardObjects verifies physical order, duplicate copies, card-name
// lookup, and shared custom-deck state are preserved for a complete zone.
func TestBuildCardObjects(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
		{ID: "pentachi", Name: "Pentachi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := buildCustomDeckState("faces.png", "back.png", 2)
	if err != nil {
		t.Fatal(err)
	}

	objects, err := buildCardObjects(
		[]string{"pentachi", "abolition", "pentachi"},
		[]int{100, 101, 100},
		1,
		state,
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 3 {
		t.Fatalf("object count = %d, want 3", len(objects))
	}

	wantNames := []string{"Pentachi", "Abolition", "Pentachi"}
	wantIDs := []int{100, 101, 100}
	for index, object := range objects {
		if object.Nickname != wantNames[index] || object.CardID != wantIDs[index] {
			t.Fatalf(
				"object %d = %q/%d, want %q/%d",
				index,
				object.Nickname,
				object.CardID,
				wantNames[index],
				wantIDs[index],
			)
		}
		if got := object.CustomDeck["1"]; got != state {
			t.Fatalf("object %d custom state = %#v, want %#v", index, got, state)
		}
	}
}

// TestBuildCardObjectsEmpty verifies an empty zone succeeds without creating
// phantom zero-valued card objects.
func TestBuildCardObjectsEmpty(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}

	objects, err := buildCardObjects(
		nil,
		nil,
		1,
		CustomDeckState{},
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 0 {
		t.Fatalf("object count = %d, want 0", len(objects))
	}
}

// TestBuildCardObjectsRejectsNilRepository verifies metadata lookup is required.
func TestBuildCardObjectsRejectsNilRepository(t *testing.T) {
	objects, err := buildCardObjects(
		[]string{"abolition"},
		[]int{100},
		1,
		CustomDeckState{},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "repository cannot be nil") {
		t.Fatalf("buildCardObjects() error = %v", err)
	}
	if objects != nil {
		t.Fatalf("error objects = %#v, want nil", objects)
	}
}

// TestBuildCardObjectsRejectsMismatchedLengths verifies position-wise input
// slices must describe the same number of physical cards.
func TestBuildCardObjectsRejectsMismatchedLengths(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}

	objects, err := buildCardObjects(
		[]string{"abolition", "abolition"},
		[]int{100},
		1,
		CustomDeckState{},
		repository,
	)
	if err == nil || !strings.Contains(err.Error(), "same number") {
		t.Fatalf("buildCardObjects() error = %v", err)
	}
	if objects != nil {
		t.Fatalf("error objects = %#v, want nil", objects)
	}
}

// TestBuildCardObjectsRejectsUnknownCard verifies lookup errors identify the
// internal ID and its one-based physical position.
func TestBuildCardObjectsRejectsUnknownCard(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := buildCustomDeckState("faces.png", "back.png", 2)
	if err != nil {
		t.Fatal(err)
	}

	objects, err := buildCardObjects(
		[]string{"abolition", "missing-card"},
		[]int{100, 101},
		1,
		state,
		repository,
	)
	if err == nil ||
		!strings.Contains(err.Error(), `"missing-card"`) ||
		!strings.Contains(err.Error(), "position 2") {
		t.Fatalf("buildCardObjects() error = %v", err)
	}
	if objects != nil {
		t.Fatalf("error objects = %#v, want nil", objects)
	}
}

// TestBuildCardObjectsWrapsConstructorError verifies malformed TTS identity
// errors retain both physical-card context and their underlying cause.
func TestBuildCardObjectsWrapsConstructorError(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, err := buildCustomDeckState("faces.png", "back.png", 1)
	if err != nil {
		t.Fatal(err)
	}

	objects, err := buildCardObjects(
		[]string{"abolition"},
		[]int{200},
		1,
		state,
		repository,
	)
	if err == nil ||
		!strings.Contains(err.Error(), `build card "abolition" at position 1`) ||
		!strings.Contains(err.Error(), "does not match deck key") {
		t.Fatalf("buildCardObjects() error = %v", err)
	}
	if objects != nil {
		t.Fatalf("error objects = %#v, want nil", objects)
	}
}

// TestBuildDeckObject verifies one complete zone combines physical IDs,
// unique sheet order, TTS metadata, contained cards, and the supplied transform.
func TestBuildDeckObject(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
		{ID: "pentachi", Name: "Pentachi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	transform := Transform{
		PosX:   -2,
		PosY:   1,
		PosZ:   3,
		RotZ:   180,
		ScaleX: 1,
		ScaleY: 1,
		ScaleZ: 1,
	}

	object, sheetIDs, err := buildDeckObject(
		"Main Deck",
		[]string{"pentachi", "abolition", "pentachi"},
		1,
		"/tts/Caster Images/main faces.png",
		"/tts/Caster Images/card back.png",
		transform,
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}

	if object.Name != "Deck" ||
		object.Nickname != "Main Deck" ||
		object.Description != "" {
		t.Fatalf(
			"object identity = %q/%q/%q",
			object.Name,
			object.Nickname,
			object.Description,
		)
	}
	if object.Transform != transform {
		t.Fatalf("Transform = %#v, want %#v", object.Transform, transform)
	}
	if !slices.Equal(object.DeckIDs, []int{100, 101, 100}) {
		t.Fatalf("DeckIDs = %v, want [100 101 100]", object.DeckIDs)
	}
	if !slices.Equal(sheetIDs, []string{"pentachi", "abolition"}) {
		t.Fatalf("sheet IDs = %v, want [pentachi abolition]", sheetIDs)
	}
	if len(object.CustomDeck) != 1 {
		t.Fatalf("CustomDeck length = %d, want 1", len(object.CustomDeck))
	}
	state, found := object.CustomDeck["1"]
	if !found {
		t.Fatalf("CustomDeck = %#v, want key 1", object.CustomDeck)
	}
	if state.FaceURL != "/tts/Caster Images/main faces.png" ||
		state.BackURL != "/tts/Caster Images/card back.png" ||
		state.NumWidth != 2 ||
		state.NumHeight != 1 ||
		!state.BackIsHidden {
		t.Fatalf("CustomDeck[1] = %#v", state)
	}
	if len(object.ContainedObjects) != 3 {
		t.Fatalf("contained object count = %d, want 3", len(object.ContainedObjects))
	}
	wantNames := []string{"Pentachi", "Abolition", "Pentachi"}
	for index, card := range object.ContainedObjects {
		if card.Nickname != wantNames[index] ||
			card.CardID != object.DeckIDs[index] {
			t.Fatalf(
				"contained card %d = %q/%d, want %q/%d",
				index,
				card.Nickname,
				card.CardID,
				wantNames[index],
				object.DeckIDs[index],
			)
		}
		if got := card.CustomDeck["1"]; got != state {
			t.Fatalf("contained card %d state = %#v, want %#v", index, got, state)
		}
	}
}

// TestBuildDeckObjectUsesDeckNamespace verifies a sideboard uses its supplied
// custom-deck key throughout the deck and every contained card.
func TestBuildDeckObjectUsesDeckNamespace(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "call-forth", Name: "Call Forth"},
	})
	if err != nil {
		t.Fatal(err)
	}

	object, _, err := buildDeckObject(
		"Sideboard",
		[]string{"call-forth", "call-forth"},
		2,
		"side.png",
		"back.png",
		unitTransform(),
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(object.DeckIDs, []int{200, 200}) {
		t.Fatalf("DeckIDs = %v, want [200 200]", object.DeckIDs)
	}
	if _, found := object.CustomDeck["2"]; !found {
		t.Fatalf("CustomDeck = %#v, want key 2", object.CustomDeck)
	}
	for index, card := range object.ContainedObjects {
		if _, found := card.CustomDeck["2"]; !found {
			t.Fatalf("contained card %d custom deck = %#v", index, card.CustomDeck)
		}
	}
}

// TestBuildDeckObjectRejectsInitialValidation verifies invalid identity, empty
// zones, and invisible transforms fail before object construction.
func TestBuildDeckObjectRejectsInitialValidation(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		nickname  string
		cardIDs   []string
		transform Transform
		wantError string
	}{
		{
			name:      "missing nickname",
			nickname:  "",
			cardIDs:   []string{"abolition"},
			transform: unitTransform(),
			wantError: "nickname",
		},
		{
			name:      "empty zone",
			nickname:  "Main Deck",
			cardIDs:   nil,
			transform: unitTransform(),
			wantError: "deck size",
		},
		{
			name:      "zero X scale",
			nickname:  "Main Deck",
			cardIDs:   []string{"abolition"},
			transform: Transform{ScaleY: 1, ScaleZ: 1},
			wantError: "scale factors",
		},
		{
			name:      "zero Y scale",
			nickname:  "Main Deck",
			cardIDs:   []string{"abolition"},
			transform: Transform{ScaleX: 1, ScaleZ: 1},
			wantError: "scale factors",
		},
		{
			name:      "zero Z scale",
			nickname:  "Main Deck",
			cardIDs:   []string{"abolition"},
			transform: Transform{ScaleX: 1, ScaleY: 1},
			wantError: "scale factors",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object, sheetIDs, err := buildDeckObject(
				test.nickname,
				test.cardIDs,
				1,
				"faces.png",
				"back.png",
				test.transform,
				repository,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("buildDeckObject() error = %v", err)
			}
			if !reflect.DeepEqual(object, DeckObject{}) || sheetIDs != nil {
				t.Fatalf("error results = %#v, %v; want zero object and nil sheet", object, sheetIDs)
			}
		})
	}
}

// TestBuildDeckObjectWrapsHelperErrors verifies each orchestration stage adds
// useful context while retaining the underlying failure.
func TestBuildDeckObjectWrapsHelperErrors(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		cardIDs    []string
		deckKey    int
		faceURL    string
		backURL    string
		repository decks.CardCatalog
		wantError  []string
	}{
		{
			name:       "ID generation",
			cardIDs:    []string{"abolition"},
			deckKey:    0,
			faceURL:    "faces.png",
			backURL:    "back.png",
			repository: repository,
			wantError:  []string{"generate deck IDs", "must be positive"},
		},
		{
			name:       "custom state",
			cardIDs:    []string{"abolition"},
			deckKey:    1,
			faceURL:    "",
			backURL:    "back.png",
			repository: repository,
			wantError:  []string{"build custom deck state", "face path"},
		},
		{
			name:       "contained cards",
			cardIDs:    []string{"missing-card"},
			deckKey:    1,
			faceURL:    "faces.png",
			backURL:    "back.png",
			repository: repository,
			wantError:  []string{"build contained cards", `"missing-card"`},
		},
		{
			name:       "nil repository",
			cardIDs:    []string{"abolition"},
			deckKey:    1,
			faceURL:    "faces.png",
			backURL:    "back.png",
			repository: nil,
			wantError:  []string{"build contained cards", "repository cannot be nil"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object, sheetIDs, err := buildDeckObject(
				"Main Deck",
				test.cardIDs,
				test.deckKey,
				test.faceURL,
				test.backURL,
				unitTransform(),
				test.repository,
			)
			if err == nil {
				t.Fatal("buildDeckObject() error = nil")
			}
			for _, fragment := range test.wantError {
				if !strings.Contains(err.Error(), fragment) {
					t.Fatalf("buildDeckObject() error = %v, want containing %q", err, fragment)
				}
			}
			if !reflect.DeepEqual(object, DeckObject{}) || sheetIDs != nil {
				t.Fatalf("error results = %#v, %v; want zero object and nil sheet", object, sheetIDs)
			}
		})
	}
}

// TestBuildSavedObject verifies a complete deck becomes separate main and side
// TTS objects with matching unique face-sheet orders.
func TestBuildSavedObject(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
		{ID: "pentachi", Name: "Pentachi"},
		{ID: "call-forth", Name: "Call Forth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		Name: "Luna/Aqua Control",
		MainDeck: []decks.DeckEntry{
			{CardID: "abolition", Quantity: 2},
			{CardID: "pentachi", Quantity: 1},
		},
		MainOrder: []string{"pentachi", "abolition", "abolition"},
		SideDeck: []decks.DeckEntry{
			{CardID: "call-forth", Quantity: 2},
		},
		SideOrder: []string{"call-forth", "call-forth"},
	}

	saved, mainSheetIDs, sideSheetIDs, err := buildSavedObject(
		deck,
		"/tts/main faces.png",
		"/tts/side faces.png",
		"/tts/card back.png",
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}

	if saved.SaveName != deck.Name {
		t.Fatalf("SaveName = %q, want %q", saved.SaveName, deck.Name)
	}
	if len(saved.ObjectStates) != 2 {
		t.Fatalf("ObjectStates length = %d, want 2", len(saved.ObjectStates))
	}
	if !slices.Equal(mainSheetIDs, []string{"pentachi", "abolition"}) {
		t.Fatalf("main sheet IDs = %v, want [pentachi abolition]", mainSheetIDs)
	}
	if !slices.Equal(sideSheetIDs, []string{"call-forth"}) {
		t.Fatalf("side sheet IDs = %v, want [call-forth]", sideSheetIDs)
	}

	mainObject := saved.ObjectStates[0]
	if mainObject.Nickname != "Luna/Aqua Control - Main Deck" {
		t.Fatalf("main nickname = %q", mainObject.Nickname)
	}
	if !slices.Equal(mainObject.DeckIDs, []int{100, 101, 101}) {
		t.Fatalf("main DeckIDs = %v, want [100 101 101]", mainObject.DeckIDs)
	}
	if mainObject.Transform.PosX != -2.5 ||
		mainObject.Transform.PosY != 1 ||
		mainObject.Transform.RotZ != 180 {
		t.Fatalf("main transform = %#v", mainObject.Transform)
	}
	if state := mainObject.CustomDeck["1"]; state.FaceURL != "/tts/main faces.png" ||
		state.BackURL != "/tts/card back.png" ||
		state.NumWidth != 2 ||
		state.NumHeight != 1 {
		t.Fatalf("main custom state = %#v", state)
	}

	sideObject := saved.ObjectStates[1]
	if sideObject.Nickname != "Luna/Aqua Control - Sideboard" {
		t.Fatalf("side nickname = %q", sideObject.Nickname)
	}
	if !slices.Equal(sideObject.DeckIDs, []int{200, 200}) {
		t.Fatalf("side DeckIDs = %v, want [200 200]", sideObject.DeckIDs)
	}
	if sideObject.Transform.PosX != 2.5 ||
		sideObject.Transform.PosY != 1 ||
		sideObject.Transform.RotZ != 180 {
		t.Fatalf("side transform = %#v", sideObject.Transform)
	}
	if state := sideObject.CustomDeck["2"]; state.FaceURL != "/tts/side faces.png" ||
		state.BackURL != "/tts/card back.png" ||
		state.NumWidth != 1 ||
		state.NumHeight != 1 {
		t.Fatalf("side custom state = %#v", state)
	}
}

// TestBuildSavedObjectOmitsEmptySideboard verifies a main-only deck does not
// require a side face path or create a phantom second object.
func TestBuildSavedObjectOmitsEmptySideboard(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		Name:      "Main Only",
		MainDeck:  []decks.DeckEntry{{CardID: "abolition", Quantity: 1}},
		MainOrder: []string{"abolition"},
	}

	saved, mainSheetIDs, sideSheetIDs, err := buildSavedObject(
		deck,
		"main.png",
		"",
		"back.png",
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.ObjectStates) != 1 {
		t.Fatalf("ObjectStates length = %d, want 1", len(saved.ObjectStates))
	}
	if !slices.Equal(mainSheetIDs, []string{"abolition"}) {
		t.Fatalf("main sheet IDs = %v, want [abolition]", mainSheetIDs)
	}
	if sideSheetIDs != nil {
		t.Fatalf("side sheet IDs = %v, want nil", sideSheetIDs)
	}
}

// TestBuildSavedObjectFallsBackToAggregateOrder verifies inconsistent stored
// copy order uses stable aggregate entry order for both JSON IDs and rendering.
func TestBuildSavedObjectFallsBackToAggregateOrder(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
		{ID: "pentachi", Name: "Pentachi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		Name: "Fallback Order",
		MainDeck: []decks.DeckEntry{
			{CardID: "abolition", Quantity: 2},
			{CardID: "pentachi", Quantity: 1},
		},
		MainOrder: []string{"invalid"},
	}

	saved, mainSheetIDs, _, err := buildSavedObject(
		deck,
		"main.png",
		"",
		"back.png",
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(mainSheetIDs, []string{"abolition", "pentachi"}) {
		t.Fatalf("main sheet IDs = %v, want [abolition pentachi]", mainSheetIDs)
	}
	if !slices.Equal(saved.ObjectStates[0].DeckIDs, []int{100, 100, 101}) {
		t.Fatalf(
			"main DeckIDs = %v, want [100 100 101]",
			saved.ObjectStates[0].DeckIDs,
		)
	}
}

// TestBuildSavedObjectRejectsInvalidInputs verifies required application data
// fails atomically before any TTS objects are returned.
func TestBuildSavedObjectRejectsInvalidInputs(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
	})
	if err != nil {
		t.Fatal(err)
	}
	validDeck := &decks.Deck{
		Name:      "Valid",
		MainDeck:  []decks.DeckEntry{{CardID: "abolition", Quantity: 1}},
		MainOrder: []string{"abolition"},
	}
	tests := []struct {
		name       string
		deck       *decks.Deck
		repository decks.CardCatalog
		wantError  string
	}{
		{name: "nil deck", deck: nil, repository: repository, wantError: "deck cannot be empty"},
		{name: "nil repository", deck: validDeck, repository: nil, wantError: "repository cannot be empty"},
		{
			name:       "empty main",
			deck:       &decks.Deck{Name: "Empty"},
			repository: repository,
			wantError:  "main deck cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			saved, mainSheetIDs, sideSheetIDs, err := buildSavedObject(
				test.deck,
				"main.png",
				"side.png",
				"back.png",
				test.repository,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("buildSavedObject() error = %v", err)
			}
			if !reflect.DeepEqual(saved, SavedObject{}) ||
				mainSheetIDs != nil ||
				sideSheetIDs != nil {
				t.Fatalf(
					"error results = %#v, %v, %v; want zero object and nil sheets",
					saved,
					mainSheetIDs,
					sideSheetIDs,
				)
			}
		})
	}
}

// TestBuildSavedObjectWrapsZoneErrors verifies main and side failures identify
// the affected zone while preserving their underlying cause.
func TestBuildSavedObjectWrapsZoneErrors(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition"},
		{ID: "call-forth", Name: "Call Forth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		Name:      "Zone Errors",
		MainDeck:  []decks.DeckEntry{{CardID: "abolition", Quantity: 1}},
		MainOrder: []string{"abolition"},
		SideDeck:  []decks.DeckEntry{{CardID: "call-forth", Quantity: 1}},
		SideOrder: []string{"call-forth"},
	}
	tests := []struct {
		name      string
		mainFace  string
		sideFace  string
		wantError []string
	}{
		{
			name:      "main face missing",
			mainFace:  "",
			sideFace:  "side.png",
			wantError: []string{"build main deck object", "face path"},
		},
		{
			name:      "side face missing",
			mainFace:  "main.png",
			sideFace:  "",
			wantError: []string{"build side deck object", "face path"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			saved, mainSheetIDs, sideSheetIDs, err := buildSavedObject(
				deck,
				test.mainFace,
				test.sideFace,
				"back.png",
				repository,
			)
			if err == nil {
				t.Fatal("buildSavedObject() error = nil")
			}
			for _, fragment := range test.wantError {
				if !strings.Contains(err.Error(), fragment) {
					t.Fatalf("buildSavedObject() error = %v, want containing %q", err, fragment)
				}
			}
			if !reflect.DeepEqual(saved, SavedObject{}) ||
				mainSheetIDs != nil ||
				sideSheetIDs != nil {
				t.Fatalf(
					"error results = %#v, %v, %v; want zero object and nil sheets",
					saved,
					mainSheetIDs,
					sideSheetIDs,
				)
			}
		})
	}
}

// TestWriteSavedObjectJSON verifies readable JSON preserves the complete TTS
// object graph and ends with a conventional newline.
func TestWriteSavedObjectJSON(t *testing.T) {
	state, err := buildCustomDeckState("main faces.png", "card back.png", 1)
	if err != nil {
		t.Fatal(err)
	}
	card, err := buildCardObject(100, "Abolition", 1, state)
	if err != nil {
		t.Fatal(err)
	}
	want := SavedObject{
		SaveName: "Luna/Aqua Control",
		ObjectStates: []DeckObject{
			{
				Name:        "Deck",
				Nickname:    "Luna/Aqua Control - Main Deck",
				Description: "",
				Transform:   unitTransform(),
				DeckIDs:     []int{100},
				CustomDeck: map[string]CustomDeckState{
					"1": state,
				},
				ContainedObjects: []CardObject{card},
			},
		},
	}

	var encoded bytes.Buffer
	if err := writeSavedObjectJSON(&encoded, want); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(encoded.Bytes()) {
		t.Fatalf("output is not valid JSON:\n%s", encoded.String())
	}
	if !strings.HasSuffix(encoded.String(), "\n") {
		t.Fatal("encoded JSON does not end with a newline")
	}
	if !strings.Contains(encoded.String(), "\n  \"SaveName\"") ||
		!strings.Contains(encoded.String(), "\n    {") {
		t.Fatalf("encoded JSON does not use two-space indentation:\n%s", encoded.String())
	}

	var got SavedObject
	if err := json.Unmarshal(encoded.Bytes(), &got); err != nil {
		t.Fatalf("decode saved object JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded object = %#v, want %#v", got, want)
	}
}

// TestWriteSavedObjectJSONRejectsInvalidInput verifies the destination, save
// name, and at least one object state are required.
func TestWriteSavedObjectJSONRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		writer    io.Writer
		object    SavedObject
		wantError string
	}{
		{
			name:      "nil writer",
			writer:    nil,
			object:    SavedObject{SaveName: "Deck", ObjectStates: []DeckObject{{}}},
			wantError: "writer cannot be nil",
		},
		{
			name:      "missing save name",
			writer:    &bytes.Buffer{},
			object:    SavedObject{ObjectStates: []DeckObject{{}}},
			wantError: "save name",
		},
		{
			name:      "nil object states",
			writer:    &bytes.Buffer{},
			object:    SavedObject{SaveName: "Deck"},
			wantError: "object states",
		},
		{
			name:      "empty object states",
			writer:    &bytes.Buffer{},
			object:    SavedObject{SaveName: "Deck", ObjectStates: []DeckObject{}},
			wantError: "object states",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := writeSavedObjectJSON(test.writer, test.object)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("writeSavedObjectJSON() error = %v", err)
			}
		})
	}
}

// TestWriteSavedObjectJSONReportsWriterFailure verifies encoder failures retain
// both TTS export context and the underlying destination error.
func TestWriteSavedObjectJSONReportsWriterFailure(t *testing.T) {
	err := writeSavedObjectJSON(
		failingJSONWriter{},
		SavedObject{
			SaveName:     "Deck",
			ObjectStates: []DeckObject{{Name: "Deck"}},
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "encode TTS saved object") ||
		!strings.Contains(err.Error(), "forced JSON writer failure") {
		t.Fatalf("writeSavedObjectJSON() error = %v", err)
	}
}

type failingJSONWriter struct{}

func (failingJSONWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("forced JSON writer failure")
}

// unitTransform returns a visible identity-scale transform for TTS objects.
func unitTransform() Transform {
	return Transform{ScaleX: 1, ScaleY: 1, ScaleZ: 1}
}

// uniqueTestCardIDs returns stable distinct identifiers for capacity tests.
func uniqueTestCardIDs(count int) []string {
	cardIDs := make([]string, count)
	for index := range cardIDs {
		cardIDs[index] = fmt.Sprintf("card-%02d", index)
	}
	return cardIDs
}
