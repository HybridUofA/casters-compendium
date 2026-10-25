package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2/storage"
)

func TestDeckNameFromSaveURI(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/tmp/My Deck v2.json", want: "My Deck v2"},
		{path: "C:/Decks/Arthur.json", want: "Arthur"},
		{path: "plain.json", want: "plain"},
	}
	for _, test := range tests {
		uri := storage.NewFileURI(test.path)
		if got := deckNameFromSaveURI(uri); got != test.want {
			t.Fatalf("deckNameFromSaveURI(%q) = %q; want %q", test.path, got, test.want)
		}
	}
}
