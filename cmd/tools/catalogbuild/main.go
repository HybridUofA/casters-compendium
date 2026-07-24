// Command catalogbuild creates the exact immutable directory tree published to
// R2. It deliberately does not contain cloud credentials or upload behavior;
// generation can be inspected and tested before a separate workflow publishes.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	deckexport "github.com/HybridUofA/casters-compendium/internal/deckbuilder/export"
)

var catalogVersionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

type options struct {
	Version     string
	BaseURL     string
	Database    string
	Images      string
	CardBack    string
	Output      string
	PublishedAt time.Time
}

func main() {
	var publishedAt string
	settings := options{}
	flag.StringVar(&settings.Version, "version", "", "immutable catalog version, for example v1")
	flag.StringVar(&settings.BaseURL, "base-url", "https://tts.casterscompendium.com", "public asset origin")
	flag.StringVar(&settings.Database, "database", "data/cards.json", "normalized card database")
	flag.StringVar(&settings.Images, "images", "data/images", "source card-image directory")
	flag.StringVar(&settings.CardBack, "card-back", "data/images/MTD-back-ver01.png", "TTS card-back image")
	flag.StringVar(&settings.Output, "output", "dist/hosted-catalog", "generated catalog root")
	flag.StringVar(&publishedAt, "published-at", "", "RFC3339 publication time; defaults to now")
	flag.Parse()

	if strings.TrimSpace(publishedAt) == "" {
		settings.PublishedAt = time.Now().UTC()
	} else {
		value, err := time.Parse(time.RFC3339, publishedAt)
		if err != nil {
			log.Fatalf("parse -published-at: %v", err)
		}
		settings.PublishedAt = value.UTC()
	}
	if err := buildCatalog(settings); err != nil {
		log.Fatal(err)
	}
	log.Printf("built hosted catalog %s under %s", settings.Version, settings.Output)
}

func buildCatalog(settings options) error {
	settings.Version = strings.TrimSpace(settings.Version)
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	if !catalogVersionPattern.MatchString(settings.Version) {
		return fmt.Errorf("catalog version must match v1, v2, and so on")
	}
	if settings.PublishedAt.IsZero() {
		return fmt.Errorf("publication time cannot be empty")
	}
	if strings.TrimSpace(settings.Output) == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	repository, err := cards.LoadFile(settings.Database)
	if err != nil {
		return err
	}
	encodedDatabase, err := distribution.EncodeCards(repository.All())
	if err != nil {
		return err
	}

	versionDirectory := filepath.Join(settings.Output, "catalog", settings.Version)
	if _, err := os.Stat(versionDirectory); err == nil {
		return fmt.Errorf("immutable catalog directory already exists: %q", versionDirectory)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect catalog output: %w", err)
	}
	imageDirectory := filepath.Join(versionDirectory, "images")
	ttsDirectory := filepath.Join(versionDirectory, "tts")
	backDirectory := filepath.Join(settings.Output, "backs")
	for _, directory := range []string{imageDirectory, ttsDirectory, backDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create output directory %q: %w", directory, err)
		}
	}

	// Hosted artwork is normalized to stable <card-id>.png paths. Current source
	// snapshots are PNG; rejecting a different extension prevents mislabeled
	// content from entering a release unnoticed.
	cardByID := make(map[string]cards.Card, len(repository.All()))
	for _, card := range repository.All() {
		cardByID[strings.TrimSpace(card.ID)] = card
	}
	sortedCardIDs, err := distribution.SortedCardIDs(repository.All())
	if err != nil {
		return err
	}
	var imageDigestInput strings.Builder
	for _, cardID := range sortedCardIDs {
		card := cardByID[cardID]
		source, found := cardimages.FindIn(settings.Images, card.ID)
		if !found {
			return fmt.Errorf("source image for card %q (%s) was not found", card.Name, card.ID)
		}
		if strings.ToLower(filepath.Ext(source)) != ".png" {
			return fmt.Errorf("source image for card %q must be PNG: %q", card.Name, source)
		}
		if strings.ContainsAny(cardID, `/\`) || filepath.Base(cardID) != cardID {
			return fmt.Errorf("card ID %q is not safe for a hosted image path", cardID)
		}
		imageBytes, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read image for card %q: %w", card.Name, err)
		}
		destination := filepath.Join(imageDirectory, cardID+".png")
		if err := writeBytesFile(destination, imageBytes); err != nil {
			return fmt.Errorf("copy image for card %q: %w", card.Name, err)
		}
		imageDigestInput.WriteString(cardID)
		imageDigestInput.WriteByte(0)
		imageDigestInput.WriteString(distribution.SHA256(imageBytes))
		imageDigestInput.WriteByte('\n')
	}
	imageCollectionSHA256 := distribution.SHA256([]byte(imageDigestInput.String()))

	backFilename := "mtd-back-v1.png"
	if err := copyFile(settings.CardBack, filepath.Join(backDirectory, backFilename)); err != nil {
		return fmt.Errorf("copy card back: %w", err)
	}
	cardBackURL := settings.BaseURL + "/backs/" + backFilename
	ttsPublicURL := settings.BaseURL + "/catalog/" + settings.Version + "/tts"
	ttsManifest, err := deckexport.GenerateHostedTTSAssets(
		ttsDirectory,
		ttsPublicURL,
		cardBackURL,
		settings.Version,
		repository,
		settings.Images,
	)
	if err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(ttsDirectory, "manifest.json"), ttsManifest); err != nil {
		return err
	}
	if err := writeBytesFile(filepath.Join(versionDirectory, "cards.json"), encodedDatabase); err != nil {
		return err
	}

	versionURL := settings.BaseURL + "/catalog/" + settings.Version
	release := distribution.ReleaseManifest{
		SchemaVersion:  distribution.SchemaVersion,
		CatalogVersion: settings.Version,
		PublishedAt:    settings.PublishedAt,
		Database: distribution.DatabaseAsset{
			URL:    versionURL + "/cards.json",
			SHA256: distribution.SHA256(encodedDatabase),
			Size:   int64(len(encodedDatabase)),
		},
		Images: distribution.ImageAssets{
			BaseURL: versionURL + "/images/",
			SHA256:  imageCollectionSHA256,
		},
		TabletopSimulator: distribution.TabletopAssets{
			ManifestURL: versionURL + "/tts/manifest.json",
			CardBackURL: cardBackURL,
		},
	}
	if err := release.Validate(); err != nil {
		return fmt.Errorf("validate release manifest: %w", err)
	}
	if err := writeJSONFile(filepath.Join(versionDirectory, "release.json"), release); err != nil {
		return err
	}

	pointer := distribution.CatalogPointer{
		SchemaVersion:  distribution.SchemaVersion,
		CatalogVersion: settings.Version,
		ReleaseURL:     versionURL + "/release.json",
	}
	if err := pointer.Validate(); err != nil {
		return fmt.Errorf("validate catalog pointer: %w", err)
	}
	if err := writeJSONFile(filepath.Join(settings.Output, "catalog", "current.json"), pointer); err != nil {
		return err
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create JSON directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		file.Close()
		return fmt.Errorf("encode %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %q: %w", path, err)
	}
	return nil
}

func writeBytesFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func copyFile(source string, destination string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := output.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	_, err = io.Copy(output, input)
	return err
}
