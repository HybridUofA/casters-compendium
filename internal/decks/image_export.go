package decks

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	deckImageWidth      = 4096
	deckImageHeight     = 3900
	deckImageColumns    = 10
	deckImageCardWidth  = 409
	deckImageCardHeight = 557
	deckImageBackFile   = cardimages.CardBackFileName
)

var deckImageBackground = color.RGBA{R: 30, G: 30, B: 40, A: 255}

// WriteDeckImage renders the main deck as a Tabletop Simulator PNG sheet.
// Face-up cards fill rows from the top-left and the standard card back occupies
// the final slot at the bottom-right.
func WriteDeckImage(
	writer io.Writer,
	deck *Deck,
	imageDirectory string,
) error {
	if writer == nil {
		return fmt.Errorf("deck image writer cannot be nil")
	}
	if deck == nil {
		return fmt.Errorf("deck cannot be nil")
	}

	mainCardIDs := cardIDsForImageExport(deck.MainDeck, deck.MainOrder)
	if len(mainCardIDs) > MaxMainDeckCards {
		return fmt.Errorf(
			"main deck has %d cards; maximum is %d",
			len(mainCardIDs),
			MaxMainDeckCards,
		)
	}
	return writeDeckCardSheet(writer, mainCardIDs, imageDirectory, "main deck")
}

// WriteSideboardImage renders the side deck as a Tabletop Simulator PNG sheet
// using the same layout and required final card-back slot as WriteDeckImage.
func WriteSideboardImage(
	writer io.Writer,
	deck *Deck,
	imageDirectory string,
) error {
	if writer == nil {
		return fmt.Errorf("sideboard image writer cannot be nil")
	}
	if deck == nil {
		return fmt.Errorf("deck cannot be nil")
	}

	sideCardIDs := cardIDsForImageExport(deck.SideDeck, deck.SideOrder)
	if len(sideCardIDs) > MaxSideDeckCards {
		return fmt.Errorf(
			"side deck has %d cards; maximum is %d",
			len(sideCardIDs),
			MaxSideDeckCards,
		)
	}
	return writeDeckCardSheet(writer, sideCardIDs, imageDirectory, "sideboard")
}

// writeDeckCardSheet renders card fronts plus the required back into a fixed-column PNG sheet.
func writeDeckCardSheet(
	writer io.Writer,
	cardIDs []string,
	imageDirectory string,
	sectionName string,
) error {
	backImage, err := openDeckImageFile(
		filepath.Join(imageDirectory, deckImageBackFile),
		"card back",
	)
	if err != nil {
		return err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, deckImageWidth, deckImageHeight))
	draw.Draw(
		canvas,
		canvas.Bounds(),
		image.NewUniform(deckImageBackground),
		image.Point{},
		draw.Src,
	)

	for index, cardID := range cardIDs {
		x := index % deckImageColumns * deckImageCardWidth
		y := index / deckImageColumns * deckImageCardHeight
		cardImage, err := openDeckImage(
			imageDirectory,
			cardID,
		)
		if err != nil {
			return fmt.Errorf("render %s card %d: %w", sectionName, index+1, err)
		}
		drawScaledDeckImage(canvas, x, y, cardImage)
	}

	drawScaledDeckImage(
		canvas,
		(deckImageColumns-1)*deckImageCardWidth,
		deckImageHeight-deckImageCardHeight,
		backImage,
	)

	if err := png.Encode(writer, canvas); err != nil {
		return fmt.Errorf("encode deck image: %w", err)
	}
	return nil
}

// ExportDeckImage writes deck as a PNG using the default card-image cache.
func ExportDeckImage(path string, deck *Deck) (err error) {
	return exportDeckCardSheet(path, deck, WriteDeckImage)
}

// ExportSideboardImage writes the side deck as a Tabletop Simulator PNG sheet
// using the default card-image cache.
func ExportSideboardImage(path string, deck *Deck) (err error) {
	return exportDeckCardSheet(path, deck, WriteSideboardImage)
}

// exportDeckCardSheet creates a destination file and delegates rendering to a zone writer.
func exportDeckCardSheet(
	path string,
	deck *Deck,
	writeImage func(io.Writer, *Deck, string) error,
) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create deck image file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close deck image file: %w", closeErr)
		}
	}()

	if err := writeImage(file, deck, cardimages.DefaultDirectory); err != nil {
		return fmt.Errorf("write deck image: %w", err)
	}
	return nil
}

// cardIDsForImageExport prefers a valid per-copy order and otherwise expands aggregate entries.
func cardIDsForImageExport(entries []DeckEntry, order []string) []string {
	if len(order) == totalCards(entries) {
		return append([]string(nil), order...)
	}

	cardIDs := make([]string, 0, totalCards(entries))
	for _, entry := range entries {
		for copyNumber := 0; copyNumber < entry.Quantity; copyNumber++ {
			cardIDs = append(cardIDs, entry.CardID)
		}
	}
	return cardIDs
}

// openDeckImage locates and decodes a card image by identifier.
func openDeckImage(imageDirectory string, cardID string) (image.Image, error) {
	path, found := cardimages.FindIn(imageDirectory, cardID)
	if !found {
		return nil, fmt.Errorf("cached image for card %q not found", cardID)
	}
	return openDeckImageFile(path, fmt.Sprintf("card %q", cardID))
}

// openDeckImageFile opens and decodes one image while adding a useful error description.
func openDeckImageFile(path string, description string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s image: %w", description, err)
	}
	defer file.Close()

	decoded, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode %s image: %w", description, err)
	}
	return decoded, nil
}

// drawScaledDeckImage scales an image smoothly into one card-sized sheet cell.
func drawScaledDeckImage(
	canvas *image.RGBA,
	x int,
	y int,
	cardImage image.Image,
) {
	destination := image.Rect(
		x,
		y,
		x+deckImageCardWidth,
		y+deckImageCardHeight,
	)
	xdraw.CatmullRom.Scale(
		canvas,
		destination,
		cardImage,
		cardImage.Bounds(),
		draw.Src,
		nil,
	)
}
