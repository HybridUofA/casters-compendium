package distribution

import (
	"strings"
	"testing"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

func validReleaseManifest() ReleaseManifest {
	return ReleaseManifest{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "v1",
		PublishedAt:    time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC),
		Database: DatabaseAsset{
			URL:    "https://tts.casterscompendium.com/catalog/v1/cards.json",
			SHA256: strings.Repeat("a", 64),
			Size:   100,
		},
		Images: ImageAssets{
			BaseURL: "https://tts.casterscompendium.com/catalog/v1/images/",
			SHA256:  strings.Repeat("b", 64),
		},
		TabletopSimulator: TabletopAssets{
			ManifestURL: "https://tts.casterscompendium.com/catalog/v1/tts/manifest.json",
			CardBackURL: "https://tts.casterscompendium.com/backs/mtd-back-v1.png",
		},
	}
}

func TestReleaseManifestValidate(t *testing.T) {
	if err := validReleaseManifest().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseManifestRejectsMutableOrMalformedFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ReleaseManifest)
	}{
		{"schema", func(value *ReleaseManifest) { value.SchemaVersion++ }},
		{"version", func(value *ReleaseManifest) { value.CatalogVersion = "" }},
		{"time", func(value *ReleaseManifest) { value.PublishedAt = time.Time{} }},
		{"database URL", func(value *ReleaseManifest) { value.Database.URL = "http://example.test/cards.json" }},
		{"database hash", func(value *ReleaseManifest) { value.Database.SHA256 = "bad" }},
		{"database size", func(value *ReleaseManifest) { value.Database.Size = 0 }},
		{"image slash", func(value *ReleaseManifest) { value.Images.BaseURL = strings.TrimSuffix(value.Images.BaseURL, "/") }},
		{"TTS URL", func(value *ReleaseManifest) { value.TabletopSimulator.ManifestURL = "relative.json" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validReleaseManifest()
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestReleaseDigestCoversDatabaseAndArtwork(t *testing.T) {
	base := validReleaseManifest()
	before, err := ReleaseDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	changedDatabase := base
	changedDatabase.Database.SHA256 = strings.Repeat("c", 64)
	changedArtwork := base
	changedArtwork.Images.SHA256 = strings.Repeat("d", 64)
	for name, manifest := range map[string]ReleaseManifest{
		"database": changedDatabase,
		"artwork":  changedArtwork,
	} {
		t.Run(name, func(t *testing.T) {
			after, err := ReleaseDigest(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if after == before {
				t.Fatalf("%s change did not alter release digest", name)
			}
		})
	}
}

func validTTSManifest() TTSManifest {
	return TTSManifest{
		SchemaVersion:  SchemaVersion,
		CatalogVersion: "v1",
		CardBackURL:    "https://tts.casterscompendium.com/backs/mtd-back-v1.png",
		Sheets: []TTSSheet{{
			DeckKey:   1,
			FaceURL:   "https://tts.casterscompendium.com/catalog/v1/tts/sheet-001.png",
			NumWidth:  2,
			NumHeight: 1,
			CardCount: 2,
		}},
		Cards: map[string]TTSCardLocation{
			"1": {DeckKey: 1, Slot: 0},
			"2": {DeckKey: 1, Slot: 1},
		},
	}
}

func TestTTSManifestValidate(t *testing.T) {
	if err := validTTSManifest().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTTSManifestRejectsInvalidReferences(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TTSManifest)
	}{
		{"duplicate key", func(value *TTSManifest) { value.Sheets = append(value.Sheets, value.Sheets[0]) }},
		{"unknown key", func(value *TTSManifest) { value.Cards["1"] = TTSCardLocation{DeckKey: 2} }},
		{"slot outside sheet", func(value *TTSManifest) { value.Cards["1"] = TTSCardLocation{DeckKey: 1, Slot: 2} }},
		{"oversized sheet", func(value *TTSManifest) { value.Sheets[0].NumWidth, value.Sheets[0].NumHeight = 10, 8 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validTTSManifest()
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestEncodeCardsIsStableAndComplete(t *testing.T) {
	cardList := []cards.Card{{ID: "1", Name: "Card", Ability: "Changed rules"}}
	encoded, err := EncodeCards(cardList)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(encoded), "\n") || !strings.Contains(string(encoded), "Changed rules") {
		t.Fatalf("unexpected canonical database: %q", encoded)
	}
	if SHA256(encoded) != SHA256(append([]byte(nil), encoded...)) {
		t.Fatal("SHA256 is not deterministic")
	}
}

func TestSortedCardIDs(t *testing.T) {
	got, err := SortedCardIDs([]cards.Card{{ID: "20"}, {ID: "3"}, {ID: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1", "20", "3"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("SortedCardIDs() = %v, want %v", got, want)
		}
	}
}
