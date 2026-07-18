package cardimages

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip lets tests provide an in-memory HTTP transport without opening sockets.
func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

// TestDownloadFromURLUsesOverride verifies bootstrap images need not use the card's source URL.
func TestDownloadFromURLUsesOverride(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(strings.NewReader("github image")),
			Request:    request,
		}, nil
	})}

	path, downloaded, err := DownloadFromURL(
		context.Background(),
		client,
		t.TempDir(),
		cards.Card{ID: "1", Name: "Card", ImageURL: "https://invalid.example/card.png"},
		"https://example.test/1.png",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Fatal("image was not marked as downloaded")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "github image" {
		t.Fatalf("downloaded data = %q", data)
	}
}
