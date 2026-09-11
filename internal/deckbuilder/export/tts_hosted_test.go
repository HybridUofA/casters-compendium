package deckexport

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

func TestGenerateHostedTTSAssetsScalesBeyondLegacyDeckKeyLimit(t *testing.T) {
	const cardCount = 379
	imageDirectory := t.TempDir()
	definitions := make([]gamecards.Card, 0, cardCount)
	for index := 1; index <= cardCount; index++ {
		cardID := fmt.Sprintf("%03d", index)
		definitions = append(definitions, gamecards.Card{ID: cardID, Name: "Card " + cardID})
		writeSolidPNG(t, filepath.Join(imageDirectory, cardID+".png"), color.White)
	}
	repository, err := cards.NewRepository(definitions)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := GenerateHostedTTSAssets(
		t.TempDir(),
		"https://tts.casterscompendium.com/catalog/v4/tts",
		"https://tts.casterscompendium.com/backs/mtd-back-v1.png",
		"v4",
		repository,
		imageDirectory,
	)
	if err != nil {
		t.Fatalf("GenerateHostedTTSAssets() error = %v", err)
	}
	if len(manifest.Sheets) != 6 {
		t.Fatalf("sheet count = %d; want 6", len(manifest.Sheets))
	}
	first := manifest.Sheets[0]
	if first.DeckKey != 1 || first.NumWidth != 10 || first.NumHeight != 7 || first.CardCount != 70 {
		t.Fatalf("first sheet = %#v; want key 1, 10x7, 70 cards", first)
	}
	last := manifest.Sheets[5]
	if last.DeckKey != 6 || last.NumWidth != 10 || last.NumHeight != 3 || last.CardCount != 29 {
		t.Fatalf("last sheet = %#v; want key 6, 10x3, 29 cards", last)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("generated manifest is invalid: %v", err)
	}
}

func testHostedManifest() distribution.TTSManifest {
	return distribution.TTSManifest{
		SchemaVersion:  distribution.SchemaVersion,
		CatalogVersion: "v1",
		CardBackURL:    "https://tts.casterscompendium.com/backs/mtd-back-v1.png",
		Sheets: []distribution.TTSSheet{
			{
				DeckKey: 1, FaceURL: "https://tts.casterscompendium.com/catalog/v1/tts/sheet-001.png",
				NumWidth: 1, NumHeight: 1, CardCount: 1,
			},
			{
				DeckKey: 2, FaceURL: "https://tts.casterscompendium.com/catalog/v1/tts/sheet-002.png",
				NumWidth: 1, NumHeight: 1, CardCount: 1,
			},
		},
		Cards: map[string]distribution.TTSCardLocation{
			"1": {DeckKey: 1, Slot: 0},
			"2": {DeckKey: 2, Slot: 0},
		},
	}
}

func testHostedRepository(t *testing.T) *cards.Repository {
	t.Helper()
	repository, err := cards.NewRepository([]gamecards.Card{
		{ID: "1", Name: "One"},
		{ID: "2", Name: "Two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func TestBuildHostedSavedObjectUsesCanonicalSheetKeys(t *testing.T) {
	deck := &decks.Deck{
		SchemaVersion: 1,
		Name:          "Shared Deck",
		MainDeck: []decks.DeckEntry{
			{CardID: "1", Quantity: 2},
			{CardID: "2", Quantity: 1},
		},
		MainOrder: []string{"1", "2", "1"},
		SideDeck:  []decks.DeckEntry{{CardID: "2", Quantity: 1}},
		SideOrder: []string{"2"},
	}
	object, err := BuildHostedSavedObject(deck, testHostedManifest(), testHostedRepository(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(object.ObjectStates) != 2 {
		t.Fatalf("ObjectStates = %d", len(object.ObjectStates))
	}
	main := object.ObjectStates[0]
	wantIDs := []int{100, 200, 100}
	for index, want := range wantIDs {
		if main.DeckIDs[index] != want {
			t.Fatalf("DeckIDs = %v, want %v", main.DeckIDs, wantIDs)
		}
	}
	if len(main.CustomDeck) != 2 ||
		main.CustomDeck["1"].FaceURL != testHostedManifest().Sheets[0].FaceURL ||
		main.CustomDeck["2"].FaceURL != testHostedManifest().Sheets[1].FaceURL {
		t.Fatalf("unexpected CustomDeck: %#v", main.CustomDeck)
	}
	if main.ContainedObjects[1].CustomDeck["2"].BackURL != testHostedManifest().CardBackURL {
		t.Fatalf("contained card did not use hosted back: %#v", main.ContainedObjects[1])
	}
}

func TestBuildHostedSavedObjectRejectsUnpublishedCard(t *testing.T) {
	deck := &decks.Deck{
		SchemaVersion: 1,
		Name:          "Missing",
		MainDeck:      []decks.DeckEntry{{CardID: "2", Quantity: 1}},
	}
	manifest := testHostedManifest()
	delete(manifest.Cards, "2")
	if _, err := BuildHostedSavedObject(deck, manifest, testHostedRepository(t)); err == nil {
		t.Fatal("BuildHostedSavedObject() accepted an unpublished card")
	}
}

func TestInstallHostedTTSDeckWritesOnlySavedObject(t *testing.T) {
	root := newTestTTSRoot(t)
	deck := &decks.Deck{
		SchemaVersion: 1,
		Name:          "Portable",
		MainDeck:      []decks.DeckEntry{{CardID: "1", Quantity: 1}},
	}
	paths, err := InstallHostedTTSDeck(root, deck, testHostedManifest(), testHostedRepository(t))
	if err != nil {
		t.Fatal(err)
	}
	if paths.MainFacePath != "" || paths.BackPath != "" || paths.ImageDirectory != "" {
		t.Fatalf("hosted installer reported local assets: %#v", paths)
	}
	data, err := os.ReadFile(paths.JSONPath)
	if err != nil {
		t.Fatal(err)
	}
	var object SavedObject
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if got := object.ObjectStates[0].CustomDeck["1"].FaceURL; got != testHostedManifest().Sheets[0].FaceURL {
		t.Fatalf("FaceURL = %q", got)
	}
	localImages := filepath.Join(root, "Mods", "Images", "CastersCompendium")
	if _, err := os.Stat(localImages); !os.IsNotExist(err) {
		t.Fatalf("hosted installer unexpectedly created %q", localImages)
	}
}
