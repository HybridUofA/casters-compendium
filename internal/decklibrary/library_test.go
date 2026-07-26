package decklibrary

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDirectoryForHome(t *testing.T) {
	got := DirectoryForHome(filepath.Join("home", "player"))
	want := filepath.Join("home", "player", "Documents", "Caster's Compendium", "Decks")
	if got != want {
		t.Fatalf("DirectoryForHome() = %q, want %q", got, want)
	}
}

func TestEnsureCreatesDeckLibrary(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "nested", "Decks")
	if err := Ensure(directory); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", directory)
	}
}

func TestEnsureRejectsEmptyDirectory(t *testing.T) {
	if err := Ensure("  "); err == nil {
		t.Fatal("Ensure() accepted an empty directory")
	}
}

func TestDiscoverReturnsSupportedDecksSorted(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"Zulu.JSON", "alpha.txt", "notes.md"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "ignored.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := Discover(directory)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{entries[0].Name, entries[1].Name}
	if want := []string{"alpha", "Zulu"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover() names = %#v, want %#v", got, want)
	}
}
