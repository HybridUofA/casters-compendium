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

	dragSource  *CardDragSource
	OnDragStart CardDragCallback
	OnDragMove  CardDragCallback
	OnDragEnd   CardDragCallback

	dragging         bool
	lastDragPosition fyne.Position
}

var _ fyne.Draggable = (*CardTile)(nil)
var _ desktop.Mouseable = (*CardTile)(nil)

type CardDragSourceKind int

const (
	DragFromSearch CardDragSourceKind = iota
	DragFromDeck
)

type CardDragSource struct {
	Kind  CardDragSourceKind
	Card  cards.Card
	Zone  decks.Zone
	Index int
}

type CardDragCallback func(
	tile *CardTile,
	source CardDragSource,
	position fyne.Position,
)

// SetDraggingVisual toggles translucency while a card is being dragged.
func (tile *CardTile) SetDraggingVisual(dragging bool) {
	if tile.image == nil {
		return
	}
	if dragging {
		tile.image.Translucency = 0.45
	} else {
		tile.image.Translucency = 0
	}
	tile.image.Refresh()
}

// EnableDrag associates a source description and lifecycle callbacks with the tile.
func (tile *CardTile) EnableDrag(
	source CardDragSource,
	onStart CardDragCallback,
	onMove CardDragCallback,
	onEnd CardDragCallback,
) {
	tile.dragSource = &source
	tile.OnDragStart = onStart
	tile.OnDragMove = onMove
	tile.OnDragEnd = onEnd
}

// Dragged starts or advances a drag using Fyne's absolute pointer position.
func (tile *CardTile) Dragged(event *fyne.DragEvent) {
	if tile.dragSource == nil {
		return
	}

	tile.lastDragPosition = event.AbsolutePosition

	if !tile.dragging {
		tile.dragging = true

		if tile.OnDragStart != nil {
			tile.OnDragStart(
				tile,
				*tile.dragSource,
				event.AbsolutePosition,
			)
		}
	}

	if tile.OnDragMove != nil {
		tile.OnDragMove(
			tile,
			*tile.dragSource,
			event.AbsolutePosition,
		)
	}
}

// DragEnd completes an active drag and resets the tile's drag state.
func (tile *CardTile) DragEnd() {
	if !tile.dragging || tile.dragSource == nil {
		return
	}
	tile.dragging = false
	if tile.OnDragEnd != nil {
		tile.OnDragEnd(
			tile,
			*tile.dragSource,
			tile.lastDragPosition,
		)
	}
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
		Card:          card,
		preferredSize: size,
		OnSelected:    onSelected,
		OnRightClick:  onRightClick,
	}

	tile.image = createCardImage(card)

	tile.ExtendBaseWidget(tile)

	return tile
}

// createCardImage loads a thumbnail when available and otherwise creates a placeholder image.
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

// CreateRenderer supplies the image-backed renderer used by the custom card widget.
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

// MouseUp dispatches secondary-click deck additions and preserves the Shift modifier.
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
