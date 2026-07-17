package deckgui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
)

var defaultCardTileSize = fyne.NewSize(130, 182)

type CardTile struct {
	widget.BaseWidget

	Card cards.Card

	preferredSize fyne.Size
	image         *canvas.Image

	OnSelected   func(cards.Card)
	OnRightClick func(cards.Card, bool)

	DragZone decks.Zone
	DragIndex int

	OnDragFinished func(
		fromZone decks.Zone,
		fromIndex int,
		position fyne.Position,
	)

	lastDragPosition fyne.Position
}

func (tile *CardTile) EnableDeckDrag(
	zone decks.Zone,
	index int,
	onFinished func(
		decks.Zone,
		int,
		fyne.Position,
	),
) {
	tile.DragZone = zone
	tile.DragIndex = index
	tile.OnDragFinished = onFinished
}

func (tile *CardTile) Dragged(
	event *fyne.DragEvent,
) {
	if tile.OnDragFinished == nil {
		return
	}

	tile.lastDragPosition =
		event.AbsolutePosition

	// Fade the source tile to show that it is moving.
	tile.image.Translucency = 0.45
	tile.image.Refresh()
}

func (tile *CardTile) DragEnd() {
	tile.image.Translucency = 0
	tile.image.Refresh()

	if tile.OnDragFinished == nil {
		return
	}

	tile.OnDragFinished(
		tile.DragZone,
		tile.DragIndex,
		tile.lastDragPosition,
	)
}

// NewCardTile creates a normally sized card tile.
//
// Use this for search results or anywhere that does not need
// a custom minimum size.
func NewCardTile(
	card cards.Card,
	onSelected func(cards.Card),
	onRightClick func(cards.Card, bool),
) *CardTile {
	return NewCardTileSized(
		card,
		defaultCardTileSize,
		onSelected,
		onRightClick,
	)
}

// NewCardTileSized creates a card tile with a custom minimum size.
//
// This is useful for the smaller main-deck and side-deck displays.
func NewCardTileSized(
	card cards.Card,
	size fyne.Size,
	onSelected func(cards.Card),
	onRightClick func(cards.Card, bool),
) *CardTile {
	tile := &CardTile{
		Card:         card,
		preferredSize: size,
		OnSelected:   onSelected,
		OnRightClick: onRightClick,
	}

	tile.image = createCardImage(card)

	tile.ExtendBaseWidget(tile)

	return tile
}

func createCardImage(card cards.Card) *canvas.Image {
	thumbnailPath, found := cardimages.FindThumbnail(card.ID)

	if found {
		cardImage := canvas.NewImageFromFile(
			thumbnailPath,
		)

		cardImage.FillMode = canvas.ImageFillContain
		cardImage.ScaleMode = canvas.ImageScaleSmooth

		return cardImage
	}

	// Fall back to the full-size cached image when a thumbnail
	// has not been generated yet.
	fullImagePath, found := cardimages.Find(card.ID)

	if found {
		cardImage := canvas.NewImageFromFile(
			fullImagePath,
		)

		cardImage.FillMode = canvas.ImageFillContain
		cardImage.ScaleMode = canvas.ImageScaleSmooth

		return cardImage
	}

	// Remain fully offline rather than requesting the remote URL.
	cardImage := canvas.NewImageFromResource(
		theme.BrokenImageIcon(),
	)

	cardImage.FillMode = canvas.ImageFillContain
	cardImage.ScaleMode = canvas.ImageScaleSmooth

	return cardImage
}

func (tile *CardTile) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tile.image)
}

// MinSize establishes a baseline size.
//
// GridWithColumns may enlarge the tile when more space is available.
func (tile *CardTile) MinSize() fyne.Size {
	return tile.preferredSize
}

// Normal left click.
func (tile *CardTile) Tapped(_ *fyne.PointEvent) {
	if tile.OnSelected != nil {
		tile.OnSelected(tile.Card)
	}
}

// Required by desktop.Mouseable.
func (tile *CardTile) MouseDown(_ *desktop.MouseEvent) {
}

func (tile *CardTile) MouseUp(event *desktop.MouseEvent) {
	if event.Button != desktop.MouseButtonSecondary {
		return
	}

	shiftHeld :=
		event.Modifier&fyne.KeyModifierShift != 0

	if tile.OnRightClick != nil {
		tile.OnRightClick(
			tile.Card,
			shiftHeld,
		)
	}
}