package deckexport

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
)

const ttsMaxTextureDimension = 8192

func writeTTSFaceSheet(
	writer io.Writer,
	sheetIDs []string,
	imageDirectory string,
) error {
	if writer == nil {
		return fmt.Errorf("TTS face-sheet writer cannot be nil")
	}
	if len(sheetIDs) == 0 {
		return fmt.Errorf("TTS sheet IDs cannot be empty")
	}
	cols, rows, err := sheetDimensions(len(sheetIDs))
	if err != nil {
		return fmt.Errorf("calculate sheet dimensions error: %w", err)
	}

	images := make([]image.Image, 0, len(sheetIDs))
	cellWidth := 0
	cellHeight := 0
	for index, cardID := range sheetIDs {
		cardImage, err := openDeckImage(imageDirectory, cardID)
		if err != nil {
			return fmt.Errorf("load TTS face %d for card %q: %w", index+1, cardID, err)
		}
		images = append(images, cardImage)
		bounds := cardImage.Bounds()
		cellWidth = max(cellWidth, bounds.Dx())
		cellHeight = max(cellHeight, bounds.Dy())
	}

	// Preserve the cached full-image resolution whenever the grid permits it.
	// TTS and common GPUs handle textures up to 8192 pixels reliably, so cap
	// each cell only when the complete sheet would exceed that boundary.
	cellWidth = min(cellWidth, ttsMaxTextureDimension/cols)
	cellHeight = min(cellHeight, ttsMaxTextureDimension/rows)
	pixelWidth := cols * cellWidth
	pixelHeight := rows * cellHeight
	canvas := image.NewRGBA(image.Rect(
		0,
		0,
		pixelWidth,
		pixelHeight,
	))
	draw.Draw(
		canvas,
		canvas.Bounds(),
		image.NewUniform(deckImageBackground),
		image.Point{},
		draw.Src,
	)
	for index, cardImage := range images {
		column := index % cols
		row := index / cols
		x := column * cellWidth
		y := row * cellHeight
		drawScaledDeckImageTo(canvas, x, y, cellWidth, cellHeight, cardImage)
	}
	if err := png.Encode(writer, canvas); err != nil {
		return fmt.Errorf("encode TTS face sheet: %w", err)
	}
	return nil
}
