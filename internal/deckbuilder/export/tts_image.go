package deckexport

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
)

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
	pixelWidth := cols * deckImageCardWidth
	pixelHeight := rows * deckImageCardHeight
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
	for index, cardID := range sheetIDs {
		column := index % cols
		row := index / cols
		x := column * deckImageCardWidth
		y := row * deckImageCardHeight
		cardImage, err := openDeckImage(imageDirectory, cardID)
		if err != nil {
			return fmt.Errorf("load TTS face %d for card %q: %w", index+1, cardID, err)
		}
		drawScaledDeckImage(canvas, x, y, cardImage)
	}
	if err := png.Encode(writer, canvas); err != nil {
		return fmt.Errorf("encode TTS face sheet: %w", err)
	}
	return nil
}
