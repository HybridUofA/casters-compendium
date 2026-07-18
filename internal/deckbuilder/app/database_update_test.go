package deckbuilder

import (
	"path/filepath"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

// TestCardListHashesMatchAcrossLocalAndRemoteRepresentations verifies canonical ordering and casing.
func TestCardListHashesMatchAcrossLocalAndRemoteRepresentations(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{
			ID:            "2",
			Name:          "Second Card",
			ImageURL:      "https://example.com/2.png",
			Expansion:     "Set B",
			IsPlaytesting: true,
		},
		{
			ID:        "1",
			Name:      "First Card Lv2",
			ImageURL:  "https://example.com/1.png",
			Expansion: "Set A",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	summaries := []speedrobo.CardResponse{
		{
			ID:          "1",
			CardKey:     "First Card lv2",
			ImageURL:    "https://example.com/1.png",
			Expansion:   "Set A",
			PlayTesting: "0",
		},
		{
			ID:          "2",
			CardKey:     "Second Card",
			ImageURL:    "https://example.com/2.png",
			Expansion:   "Set B",
			PlayTesting: "1",
		},
	}

	localHash, err := hashRepositoryCardList(repository)
	if err != nil {
		t.Fatal(err)
	}
	remoteHash, err := hashRemoteCardList(summaries)
	if err != nil {
		t.Fatal(err)
	}
	if localHash != remoteHash {
		t.Fatalf("local hash %q does not match remote hash %q", localHash, remoteHash)
	}
}

// TestCardListHashChangesWithRemoteList verifies meaningful summary changes alter the digest.
func TestCardListHashChangesWithRemoteList(t *testing.T) {
	summaries := []speedrobo.CardResponse{{
		ID:          "1",
		CardKey:     "Card",
		ImageURL:    "https://example.com/old.png",
		Expansion:   "Set",
		PlayTesting: "0",
	}}
	before, err := hashRemoteCardList(summaries)
	if err != nil {
		t.Fatal(err)
	}
	summaries[0].ImageURL = "https://example.com/new.png"
	after, err := hashRemoteCardList(summaries)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("card-list hash did not change after the image URL changed")
	}
}

// TestWriteAndReadCardListHash verifies the on-disk digest round trip.
func TestWriteAndReadCardListHash(t *testing.T) {
	hash, err := hashCardListEntries([]cardListHashEntry{{ID: "1", Name: "card"}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cardlist.sha256")
	if err := writeCardListHash(path, hash); err != nil {
		t.Fatal(err)
	}
	loaded, err := readCardListHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != hash {
		t.Fatalf("loaded hash = %q, want %q", loaded, hash)
	}
}
