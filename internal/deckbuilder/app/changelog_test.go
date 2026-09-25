package deckbuilder

import (
	"testing"

	releasenotes "github.com/HybridUofA/casters-compendium/docs/releases"
)

func TestBundledChangelogStartsWithApplicationRelease(t *testing.T) {
	notes, err := releasenotes.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) == 0 {
		t.Fatal("bundled changelog contains no releases")
	}
	want := "v" + applicationVersion
	if notes[0].Version != want {
		t.Fatalf("newest bundled changelog version = %q, want %q", notes[0].Version, want)
	}
}
