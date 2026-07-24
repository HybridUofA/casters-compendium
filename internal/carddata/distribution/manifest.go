// Package distribution defines the public, versioned catalog format served by
// tts.casterscompendium.com. Keeping this contract outside the UI and exporter
// lets future applications consume the same catalog without depending on Fyne.
package distribution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

const SchemaVersion = 1

// CatalogPointer is the only mutable catalog object. Publishers upload it last
// so clients never discover a partially uploaded immutable release.
type CatalogPointer struct {
	SchemaVersion  int    `json:"schemaVersion"`
	CatalogVersion string `json:"catalogVersion"`
	ReleaseURL     string `json:"releaseURL"`
}

// ReleaseManifest ties one immutable card database to its artwork and TTS
// assets. Database hashes cover every normalized field, including rules text.
type ReleaseManifest struct {
	SchemaVersion     int            `json:"schemaVersion"`
	CatalogVersion    string         `json:"catalogVersion"`
	PublishedAt       time.Time      `json:"publishedAt"`
	Database          DatabaseAsset  `json:"database"`
	Images            ImageAssets    `json:"images"`
	TabletopSimulator TabletopAssets `json:"tabletopSimulator"`
}

type DatabaseAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type ImageAssets struct {
	BaseURL string `json:"baseURL"`
	SHA256  string `json:"sha256"`
}

type TabletopAssets struct {
	ManifestURL string `json:"manifestURL"`
	CardBackURL string `json:"cardBackURL"`
}

// TTSManifest maps stable internal card IDs onto reusable canonical sheets.
// DeckKey and Slot map directly onto Tabletop Simulator's DeckID convention:
// DeckID = DeckKey*100 + Slot.
type TTSManifest struct {
	SchemaVersion  int                        `json:"schemaVersion"`
	CatalogVersion string                     `json:"catalogVersion"`
	CardBackURL    string                     `json:"cardBackURL"`
	Sheets         []TTSSheet                 `json:"sheets"`
	Cards          map[string]TTSCardLocation `json:"cards"`
}

type TTSSheet struct {
	DeckKey   int    `json:"deckKey"`
	FaceURL   string `json:"faceURL"`
	NumWidth  int    `json:"numWidth"`
	NumHeight int    `json:"numHeight"`
	CardCount int    `json:"cardCount"`
}

type TTSCardLocation struct {
	DeckKey int `json:"deckKey"`
	Slot    int `json:"slot"`
}

func (pointer CatalogPointer) Validate() error {
	if pointer.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported catalog pointer schema version %d", pointer.SchemaVersion)
	}
	if strings.TrimSpace(pointer.CatalogVersion) == "" {
		return fmt.Errorf("catalog version cannot be empty")
	}
	if err := validateHTTPSURL("release URL", pointer.ReleaseURL); err != nil {
		return err
	}
	return nil
}

func (manifest ReleaseManifest) Validate() error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported release schema version %d", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.CatalogVersion) == "" {
		return fmt.Errorf("catalog version cannot be empty")
	}
	if manifest.PublishedAt.IsZero() {
		return fmt.Errorf("published time cannot be empty")
	}
	if err := validateHTTPSURL("database URL", manifest.Database.URL); err != nil {
		return err
	}
	if err := validateSHA256(manifest.Database.SHA256); err != nil {
		return fmt.Errorf("database SHA-256: %w", err)
	}
	if manifest.Database.Size <= 0 {
		return fmt.Errorf("database size must be positive")
	}
	if err := validateHTTPSURL("image base URL", manifest.Images.BaseURL); err != nil {
		return err
	}
	if !strings.HasSuffix(manifest.Images.BaseURL, "/") {
		return fmt.Errorf("image base URL must end with a slash")
	}
	if err := validateSHA256(manifest.Images.SHA256); err != nil {
		return fmt.Errorf("image collection SHA-256: %w", err)
	}
	if err := validateHTTPSURL("TTS manifest URL", manifest.TabletopSimulator.ManifestURL); err != nil {
		return err
	}
	if err := validateHTTPSURL("TTS card-back URL", manifest.TabletopSimulator.CardBackURL); err != nil {
		return err
	}
	return nil
}

// ReleaseDigest is the compact installed-state marker. Including the database
// and image collection detects artwork-only corrections between catalog
// versions without hashing hundreds of local files at every startup.
func ReleaseDigest(manifest ReleaseManifest) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	return SHA256([]byte(
		manifest.CatalogVersion + "\n" +
			strings.ToLower(manifest.Database.SHA256) + "\n" +
			strings.ToLower(manifest.Images.SHA256) + "\n",
	)), nil
}

func (manifest TTSManifest) Validate() error {
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported TTS schema version %d", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.CatalogVersion) == "" {
		return fmt.Errorf("catalog version cannot be empty")
	}
	if err := validateHTTPSURL("TTS card-back URL", manifest.CardBackURL); err != nil {
		return err
	}
	if len(manifest.Sheets) == 0 {
		return fmt.Errorf("TTS sheets cannot be empty")
	}
	if len(manifest.Cards) == 0 {
		return fmt.Errorf("TTS card mappings cannot be empty")
	}

	sheets := make(map[int]TTSSheet, len(manifest.Sheets))
	for index, sheet := range manifest.Sheets {
		if sheet.DeckKey <= 0 || sheet.DeckKey > 99 {
			return fmt.Errorf("sheet %d has invalid deck key %d", index+1, sheet.DeckKey)
		}
		if _, duplicate := sheets[sheet.DeckKey]; duplicate {
			return fmt.Errorf("duplicate TTS deck key %d", sheet.DeckKey)
		}
		if err := validateHTTPSURL("TTS face URL", sheet.FaceURL); err != nil {
			return fmt.Errorf("sheet %d: %w", index+1, err)
		}
		if sheet.NumWidth <= 0 || sheet.NumHeight <= 0 {
			return fmt.Errorf("sheet %d dimensions must be positive", index+1)
		}
		capacity := sheet.NumWidth * sheet.NumHeight
		if sheet.CardCount <= 0 || sheet.CardCount > capacity || capacity > 70 {
			return fmt.Errorf("sheet %d has invalid card count or capacity", index+1)
		}
		sheets[sheet.DeckKey] = sheet
	}

	for cardID, location := range manifest.Cards {
		if strings.TrimSpace(cardID) == "" {
			return fmt.Errorf("TTS card mapping contains an empty card ID")
		}
		sheet, found := sheets[location.DeckKey]
		if !found {
			return fmt.Errorf("card %q references unknown deck key %d", cardID, location.DeckKey)
		}
		if location.Slot < 0 || location.Slot >= sheet.CardCount {
			return fmt.Errorf("card %q has invalid slot %d", cardID, location.Slot)
		}
	}
	return nil
}

// EncodeCards produces the canonical bytes hashed by release manifests and
// installed locally by clients. A single encoder prevents formatting-only hash
// changes between the publisher and desktop application.
func EncodeCards(cardList []cards.Card) ([]byte, error) {
	if len(cardList) == 0 {
		return nil, fmt.Errorf("card list cannot be empty")
	}
	encoded, err := json.MarshalIndent(cardList, "", " ")
	if err != nil {
		return nil, fmt.Errorf("encode card database: %w", err)
	}
	return append(encoded, '\n'), nil
}

func SHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// SortedCardIDs makes sheet placement deterministic across builds.
func SortedCardIDs(cardList []cards.Card) ([]string, error) {
	ids := make([]string, 0, len(cardList))
	seen := make(map[string]struct{}, len(cardList))
	for index, card := range cardList {
		id := strings.TrimSpace(card.ID)
		if id == "" {
			return nil, fmt.Errorf("card at index %d has no ID", index)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("duplicate card ID %q", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func validateHTTPSURL(label string, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute HTTPS URL", label)
	}
	return nil
}

func validateSHA256(value string) error {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("must contain a 64-character hexadecimal digest")
	}
	return nil
}
