package deckbuilder

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dataassets "github.com/HybridUofA/casters-compendium/data"
	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

func TestLoadOrDownloadCardDatabaseUsesHostedReleaseBeforeUpstream(t *testing.T) {
	client, release := hostedTestClient(t)
	restore := hostedCatalogClientFactory
	hostedCatalogClientFactory = func() (distribution.Client, error) { return client, nil }
	t.Cleanup(func() { hostedCatalogClientFactory = restore })

	path := filepath.Join(t.TempDir(), "cards.json")
	repository, snapshot, err := loadOrDownloadCardDatabase(
		context.Background(),
		path,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := repository.FindByID("1"); !found {
		t.Fatal("hosted repository was not installed")
	}
	if snapshot == nil || snapshot.imageURL == nil {
		t.Fatal("hosted image resolver was not returned")
	}
	want := release.Images.BaseURL + "1.png"
	if got := snapshot.imageURL(gamecards.Card{ID: "1"}); got != want {
		t.Fatalf("image URL = %q, want %q", got, want)
	}
	if _, err := cards.LoadFile(path); err != nil {
		t.Fatalf("installed database is invalid: %v", err)
	}
}

func TestInstallPreferredTTSDeckUsesHostedManifest(t *testing.T) {
	client, _ := hostedTestClient(t)
	restore := hostedCatalogClientFactory
	hostedCatalogClientFactory = func() (distribution.Client, error) { return client, nil }
	t.Cleanup(func() { hostedCatalogClientFactory = restore })

	root := t.TempDir()
	for _, child := range []string{"Mods", "Saves"} {
		if err := os.MkdirAll(filepath.Join(root, child), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := cards.NewRepository([]gamecards.Card{{ID: "1", Name: "One"}})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		SchemaVersion: 1,
		Name:          "Online",
		MainDeck:      []decks.DeckEntry{{CardID: "1", Quantity: 1}},
	}
	paths, hosted, fallbackReason, err := installPreferredTTSDeck(
		context.Background(), root, deck, repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !hosted || fallbackReason != nil {
		t.Fatalf("hosted = %t, fallbackReason = %v", hosted, fallbackReason)
	}
	data, err := os.ReadFile(paths.JSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "https://tts.casterscompendium.com/catalog/v1/tts/sheet-001.png") {
		t.Fatalf("saved object does not contain hosted face URL: %s", data)
	}
}

func TestInstallPreferredTTSDeckFallsBackToLocalAssets(t *testing.T) {
	restoreFactory := hostedCatalogClientFactory
	hostedCatalogClientFactory = func() (distribution.Client, error) {
		return distribution.Client{}, errors.New("catalog offline")
	}
	t.Cleanup(func() { hostedCatalogClientFactory = restoreFactory })

	root := t.TempDir()
	for _, child := range []string{"Mods", "Saves"} {
		if err := os.MkdirAll(filepath.Join(root, child), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	imageDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(imageDirectory, "1.png"), dataassets.CardBackPNG, 0o644); err != nil {
		t.Fatal(err)
	}
	oldImages, oldThumbnails := cardimages.DefaultDirectory, cardimages.ThumbnailDirectory
	cardimages.ConfigureDirectories(imageDirectory, oldThumbnails)
	t.Cleanup(func() { cardimages.ConfigureDirectories(oldImages, oldThumbnails) })

	repository, err := cards.NewRepository([]gamecards.Card{{ID: "1", Name: "One"}})
	if err != nil {
		t.Fatal(err)
	}
	deck := &decks.Deck{
		SchemaVersion: 1,
		Name:          "Fallback",
		MainDeck:      []decks.DeckEntry{{CardID: "1", Quantity: 1}},
	}
	paths, hosted, fallbackReason, err := installPreferredTTSDeck(
		context.Background(), root, deck, repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if hosted || fallbackReason == nil {
		t.Fatalf("hosted = %t, fallbackReason = %v", hosted, fallbackReason)
	}
	for _, path := range []string{paths.JSONPath, paths.MainFacePath, paths.BackPath} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("local fallback did not write %q: %v", path, err)
		}
	}
}

func hostedTestClient(t *testing.T) (distribution.Client, distribution.ReleaseManifest) {
	t.Helper()
	cardList := []gamecards.Card{{ID: "1", Name: "One"}}
	database, err := distribution.EncodeCards(cardList)
	if err != nil {
		t.Fatal(err)
	}
	pointer := distribution.CatalogPointer{
		SchemaVersion: distribution.SchemaVersion, CatalogVersion: "v1",
		ReleaseURL: "https://assets.test/catalog/v1/release.json",
	}
	release := distribution.ReleaseManifest{
		SchemaVersion: distribution.SchemaVersion, CatalogVersion: "v1",
		PublishedAt: time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC),
		Database: distribution.DatabaseAsset{
			URL:    "https://assets.test/catalog/v1/cards.json",
			SHA256: distribution.SHA256(database), Size: int64(len(database)),
		},
		Images: distribution.ImageAssets{
			BaseURL: "https://tts.casterscompendium.com/catalog/v1/images/",
			SHA256:  strings.Repeat("b", 64),
		},
		TabletopSimulator: distribution.TabletopAssets{
			ManifestURL: "https://assets.test/catalog/v1/tts/manifest.json",
			CardBackURL: "https://tts.casterscompendium.com/backs/mtd-back-v1.png",
		},
	}
	tts := distribution.TTSManifest{
		SchemaVersion: distribution.SchemaVersion, CatalogVersion: "v1",
		CardBackURL: release.TabletopSimulator.CardBackURL,
		Sheets: []distribution.TTSSheet{{
			DeckKey:  1,
			FaceURL:  "https://tts.casterscompendium.com/catalog/v1/tts/sheet-001.png",
			NumWidth: 1, NumHeight: 1, CardCount: 1,
		}},
		Cards: map[string]distribution.TTSCardLocation{
			"1": {DeckKey: 1, Slot: 0},
		},
	}
	responses := map[string][]byte{
		"https://assets.test/catalog/current.json":         hostedJSON(t, pointer),
		"https://assets.test/catalog/v1/release.json":      hostedJSON(t, release),
		"https://assets.test/catalog/v1/cards.json":        database,
		"https://assets.test/catalog/v1/tts/manifest.json": hostedJSON(t, tts),
	}
	return distribution.Client{
		PointerURL: "https://assets.test/catalog/current.json",
		HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			data, found := responses[request.URL.String()]
			status := http.StatusOK
			if !found {
				status = http.StatusNotFound
			}
			return &http.Response{
				StatusCode: status,
				Status:     http.StatusText(status),
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(data))),
				Request:    request,
			}, nil
		})},
	}, release
}

func hostedJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
