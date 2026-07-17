package cardimages

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	ThumbnailWidth  = 140
	ThumbnailHeight = 196
)

var ThumbnailDirectory = "data/thumbnails"

// FindThumbnail returns the JPEG thumbnail path for cardID when it exists.
func FindThumbnail(cardID string) (string, bool) {
	cardID = sanitizeID(cardID)
	if cardID == "" {
		return "", false
	}

	path := filepath.Join(ThumbnailDirectory, cardID+".jpg")

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}

	return path, true
}

// CreateThumbnail scales a cached full card image into the configured thumbnail directory.
func CreateThumbnail(cardID string) (string, error) {
	cardID = sanitizeID(cardID)
	if cardID == "" {
		return "", fmt.Errorf("card ID cannot be empty")
	}

	sourcePath, found := Find(cardID)
	if !found {
		return "", fmt.Errorf("full image for card %q not found", cardID)
	}

	if err := os.MkdirAll(ThumbnailDirectory, 0o755); err != nil {
		return "", fmt.Errorf("create thumbnail directory: %w", err)
	}

	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("open image source: %w", err)
	}

	defer inputFile.Close()

	sourceImage, _, err := image.Decode(inputFile)
	if err != nil {
		return "", fmt.Errorf("decode source image: %w", err)
	}

	thumb := image.NewRGBA(image.Rect(0, 0, ThumbnailWidth, ThumbnailHeight))

	draw.CatmullRom.Scale(thumb, thumb.Bounds(), sourceImage, sourceImage.Bounds(), draw.Over, nil)

	outputPath := filepath.Join(ThumbnailDirectory, cardID+".jpg")

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create thumbnail: %w", err)
	}
	defer outputFile.Close()

	err = jpeg.Encode(outputFile, thumb, &jpeg.Options{Quality: 85})
	if err != nil {
		return "", fmt.Errorf("encode thumbnail: %w", err)
	}
	return outputPath, nil
}
