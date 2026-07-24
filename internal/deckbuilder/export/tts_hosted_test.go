package deckexport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

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
