// Package cardimages manages the shared local artwork and thumbnail cache.
package cardimages

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

const maxImageBytes = 25 << 20

var DefaultDirectory = "data/images"

// ConfigureDirectories selects the runtime locations used for full images and thumbnails.
func ConfigureDirectories(imageDirectory string, thumbnailDirectory string) {
	DefaultDirectory = imageDirectory
	ThumbnailDirectory = thumbnailDirectory
}

// Find returns the cached full-image path for cardID in the configured directory.
func Find(cardID string) (string, bool) {
	return FindIn(DefaultDirectory, cardID)
}

// FindIn returns the first cached image for cardID in an explicit directory.
func FindIn(directory string, cardID string) (string, bool) {
	cardID = sanitizeID(cardID)

	if cardID == "" {
		return "", false
	}

	pattern := filepath.Join(
		directory,
		cardID+".*",
	)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", false
	}

	for _, match := range matches {
		info, statErr := os.Stat(match)
		if statErr != nil || info.IsDir() {
			continue
		}

		return match, true
	}

	return "", false
}

// Download caches a card image atomically, returning false when it was already present.
func Download(
	ctx context.Context,
	client *http.Client,
	directory string,
	card cards.Card,
) (string, bool, error) {
	return DownloadFromURL(ctx, client, directory, card, card.ImageURL)
}

// DownloadFromURL caches a card image from an explicit source URL.
func DownloadFromURL(
	ctx context.Context,
	client *http.Client,
	directory string,
	card cards.Card,
	imageURL string,
) (string, bool, error) {
	if client == nil {
		return "", false, fmt.Errorf(
			"HTTP client cannot be nil",
		)
	}

	cardID := sanitizeID(card.ID)
	if cardID == "" {
		return "", false, fmt.Errorf(
			"card %q has no ID",
			card.Name,
		)
	}

	if existingPath, found := FindIn(
		directory,
		cardID,
	); found {
		return existingPath, false, nil
	}

	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return "", false, fmt.Errorf(
			"card %q has no image URL",
			card.Name,
		)
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", false, fmt.Errorf(
			"create image directory: %w",
			err,
		)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		imageURL,
		nil,
	)
	if err != nil {
		return "", false, fmt.Errorf(
			"create request for %q: %w",
			card.Name,
			err,
		)
	}

	request.Header.Set(
		"User-Agent",
		"CastersCompendium/0.1",
	)

	response, err := client.Do(request)
	if err != nil {
		return "", false, fmt.Errorf(
			"download %q: %w",
			card.Name,
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return "", false, fmt.Errorf(
			"download %q: server returned %s",
			card.Name,
			response.Status,
		)
	}

	extension := imageExtension(
		imageURL,
		response.Header.Get("Content-Type"),
	)

	destination := filepath.Join(
		directory,
		cardID+extension,
	)

	temporaryFile, err := os.CreateTemp(
		directory,
		".card-image-*.part",
	)
	if err != nil {
		return "", false, fmt.Errorf(
			"create temporary file for %q: %w",
			card.Name,
			err,
		)
	}

	temporaryPath := temporaryFile.Name()

	// Remove an unfinished temporary file after an error.
	defer os.Remove(temporaryPath)

	limitedReader := io.LimitReader(
		response.Body,
		maxImageBytes+1,
	)

	written, err := io.Copy(
		temporaryFile,
		limitedReader,
	)
	if err != nil {
		temporaryFile.Close()

		return "", false, fmt.Errorf(
			"save image for %q: %w",
			card.Name,
			err,
		)
	}

	if written > maxImageBytes {
		temporaryFile.Close()

		return "", false, fmt.Errorf(
			"image for %q exceeds 25 MiB",
			card.Name,
		)
	}

	if err := temporaryFile.Close(); err != nil {
		return "", false, fmt.Errorf(
			"close image for %q: %w",
			card.Name,
			err,
		)
	}

	if err := os.Rename(
		temporaryPath,
		destination,
	); err != nil {
		return "", false, fmt.Errorf(
			"finish image for %q: %w",
			card.Name,
			err,
		)
	}

	return destination, true, nil
}

// imageExtension selects a supported filename extension from the URL or response media type.
func imageExtension(
	rawURL string,
	contentType string,
) string {
	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		extension := strings.ToLower(
			filepath.Ext(parsedURL.Path),
		)

		switch extension {
		case ".jpg", ".jpeg", ".png",
			".webp", ".gif":
			return extension
		}
	}

	mediaType := strings.ToLower(
		strings.TrimSpace(
			strings.Split(contentType, ";")[0],
		),
	)

	switch mediaType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		// Most of the card images should already have an
		// extension in their URL. This is only a fallback.
		return ".img"
	}
}

// sanitizeID converts a card identifier into a safe cross-platform filename component.
func sanitizeID(cardID string) string {
	cardID = strings.TrimSpace(cardID)

	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
	)

	return replacer.Replace(cardID)
}
