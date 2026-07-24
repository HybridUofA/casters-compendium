package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

func TestBuildCatalogCreatesCompleteVersionedTree(t *testing.T) {
	root := t.TempDir()
	imageDirectory := filepath.Join(root, "images")
	if err := os.MkdirAll(imageDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"1.png", "2.png", "back.png"} {
		writeTestPNG(t, filepath.Join(imageDirectory, name))
	}
	databasePath := filepath.Join(root, "cards.json")
	cardList := []cards.Card{
		{ID: "2", Name: "Two", ImageURL: "https://example.test/2.png"},
		{ID: "1", Name: "One", ImageURL: "https://example.test/1.png"},
	}
	encoded, err := json.Marshal(cardList)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(root, "output")
	err = buildCatalog(options{
		Version:     "v1",
		BaseURL:     "https://tts.casterscompendium.com",
		Database:    databasePath,
		Images:      imageDirectory,
		CardBack:    filepath.Join(imageDirectory, "back.png"),
		Output:      output,
		PublishedAt: time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	required := []string{
		"backs/mtd-back-v1.png",
		"catalog/current.json",
		"catalog/v1/cards.json",
		"catalog/v1/images/1.png",
		"catalog/v1/images/2.png",
		"catalog/v1/release.json",
		"catalog/v1/tts/manifest.json",
		"catalog/v1/tts/sheet-001.png",
	}
	for _, relative := range required {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(relative))); err != nil {
			t.Errorf("%s was not generated: %v", relative, err)
		}
	}

	var manifest distribution.TTSManifest
	readJSONFile(t, filepath.Join(output, "catalog/v1/tts/manifest.json"), &manifest)
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	if manifest.Cards["1"].Slot != 0 || manifest.Cards["2"].Slot != 1 {
		t.Fatalf("card placement is not stable: %#v", manifest.Cards)
	}

	var release distribution.ReleaseManifest
	readJSONFile(t, filepath.Join(output, "catalog/v1/release.json"), &release)
	databaseBytes, err := os.ReadFile(filepath.Join(output, "catalog/v1/cards.json"))
	if err != nil {
		t.Fatal(err)
	}
	if release.Database.SHA256 != distribution.SHA256(databaseBytes) ||
		release.Database.Size != int64(len(databaseBytes)) {
		t.Fatalf("database integrity metadata does not match output: %#v", release.Database)
	}
}

func TestBuildCatalogRefusesToOverwriteImmutableVersion(t *testing.T) {
	root := t.TempDir()
	versionDirectory := filepath.Join(root, "catalog", "v1")
	if err := os.MkdirAll(versionDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	err := buildCatalog(options{
		Version:     "v1",
		BaseURL:     "https://tts.casterscompendium.com",
		Database:    "unused",
		Output:      root,
		PublishedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("buildCatalog() unexpectedly overwrote an immutable version")
	}
}

func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 4, 6))
	for y := range 6 {
		for x := range 4 {
			canvas.Set(x, y, color.RGBA{R: 100, G: 20, B: 200, A: 255})
		}
	}
	if err := png.Encode(file, canvas); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func readJSONFile(t *testing.T, path string, destination any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		t.Fatal(err)
	}
}
