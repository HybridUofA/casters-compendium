// Package decklibrary manages the user's discoverable deck-file collection.
//
// Deck serialization remains in deckio. This package only owns the default
// location and filesystem discovery so GUI code does not need to scan folders.
package decklibrary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	applicationDirectory = "Caster's Compendium"
	decksDirectory       = "Decks"
)

// Entry describes one supported deck file in the user's deck library.
type Entry struct {
	Name string
	Path string
}

// DefaultDirectory returns the user-facing deck library beneath Documents.
func DefaultDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return DirectoryForHome(home), nil
}

// DirectoryForHome constructs the default path for a known home directory.
// It is exported to keep platform-neutral path behavior straightforward to test.
func DirectoryForHome(home string) string {
	return filepath.Join(home, "Documents", applicationDirectory, decksDirectory)
}

// Ensure creates the deck library and any missing parent directories.
func Ensure(directory string) error {
	if strings.TrimSpace(directory) == "" {
		return fmt.Errorf("deck library directory cannot be empty")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create deck library: %w", err)
	}
	return nil
}

// Discover lists editable JSON decks and compatible text decklists.
func Discover(directory string) ([]Entry, error) {
	items, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read deck library: %w", err)
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(item.Name()))
		if extension != ".json" && extension != ".txt" {
			continue
		}
		entries = append(entries, Entry{
			Name: strings.TrimSuffix(item.Name(), filepath.Ext(item.Name())),
			Path: filepath.Join(directory, item.Name()),
		})
	}

	sort.Slice(entries, func(left, right int) bool {
		return strings.ToLower(entries[left].Name) < strings.ToLower(entries[right].Name)
	})
	return entries, nil
}
