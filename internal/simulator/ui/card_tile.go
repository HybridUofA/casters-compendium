package ui

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	dataassets "github.com/HybridUofA/casters-compendium/data"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// CardTile is the simulator's presentation-only card widget. Hover previews
// are desktop-friendly, while tapping provides equivalent touch behavior.
type CardTile struct {
	widget.BaseWidget

	View            simulatorview.CardView
	Card            cards.Card
	baseSize        fyne.Size
	size            fyne.Size
	image           *canvas.Image
	uprightImage    *canvas.Image
	selectionBorder *canvas.Rectangle
	OnPreview       func(cards.Card)
	OnHiddenPreview func()
	OnActivate      func()
}

var _ desktop.Hoverable = (*CardTile)(nil)

var (
	sidewaysCardBackOnce sync.Once
	sidewaysCardBack     image.Image
	cardImageCache       sync.Map
)

func NewCardTile(
	cardView simulatorview.CardView,
	card cards.Card,
	size fyne.Size,
	onPreview func(cards.Card),
	onHiddenPreview func(),
) *CardTile {
	tile := &CardTile{
		View:            cardView,
		Card:            card,
		baseSize:        size,
		size:            size,
		OnPreview:       onPreview,
		OnHiddenPreview: onHiddenPreview,
	}
	if cardView.ShowFace {
		tile.image = simulatorCardImage(card)
	} else {
		tile.image = cardBackImage()
	}
	tile.uprightImage = tile.image
	tile.selectionBorder = canvas.NewRectangle(color.Transparent)
	tile.selectionBorder.StrokeColor = theme.Color(theme.ColorNamePrimary)
	tile.selectionBorder.StrokeWidth = 4
	tile.selectionBorder.Hide()
	tile.ExtendBaseWidget(tile)
	return tile
}

// SetSelected displays whether this card is included in the pending UI choice.
func (tile *CardTile) SetSelected(selected bool) {
	if selected {
		tile.selectionBorder.Show()
	} else {
		tile.selectionBorder.Hide()
	}
	tile.selectionBorder.Refresh()
}

// SetSideways rotates a concealed card back clockwise. Portrait tiles swap
// their layout dimensions so a Rested card retains the same visual scale;
// already-landscape tiles such as Orbs retain their requested dimensions.
// This is presentation state only and does not change authoritative state.
func (tile *CardTile) SetSideways(sideways bool) {
	tile.size = tile.baseSize
	tile.image = tile.uprightImage
	if !sideways || tile.View.ShowFace {
		tile.Refresh()
		return
	}
	tile.image = sidewaysCardBackImage()
	if tile.baseSize.Height > tile.baseSize.Width {
		tile.size = fyne.NewSize(tile.baseSize.Height, tile.baseSize.Width)
	}
	tile.Refresh()
}

func simulatorCardImage(card cards.Card) *canvas.Image {
	if path, found := cardimages.FindThumbnail(card.ID); found {
		if cached, loaded := cachedCardImage(path); loaded {
			return newSimulatorCanvasImage(cached)
		}
	}
	if path, found := cardimages.Find(card.ID); found {
		if cached, loaded := cachedCardImage(path); loaded {
			return newSimulatorCanvasImage(cached)
		}
	}
	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = canvas.ImageScaleSmooth
	return image
}

func cachedCardImage(path string) (image.Image, bool) {
	if cached, exists := cardImageCache.Load(path); exists {
		return cached.(image.Image), true
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	if err != nil {
		return nil, false
	}
	actual, _ := cardImageCache.LoadOrStore(path, decoded)
	return actual.(image.Image), true
}

func newSimulatorCanvasImage(source image.Image) *canvas.Image {
	result := canvas.NewImageFromImage(source)
	result.FillMode = canvas.ImageFillContain
	result.ScaleMode = canvas.ImageScaleSmooth
	return result
}

func cardBackImage() *canvas.Image {
	resource := fyne.NewStaticResource("caster-card-back.png", dataassets.CardBackPNG)
	image := canvas.NewImageFromResource(resource)
	image.FillMode = canvas.ImageFillContain
	image.ScaleMode = canvas.ImageScaleSmooth
	return image
}

func sidewaysCardBackImage() *canvas.Image {
	sidewaysCardBackOnce.Do(func() {
		source, _, err := image.Decode(bytes.NewReader(dataassets.CardBackPNG))
		if err != nil {
			return
		}
		bounds := source.Bounds()
		rotated := image.NewNRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for sourceY := bounds.Min.Y; sourceY < bounds.Max.Y; sourceY++ {
			for sourceX := bounds.Min.X; sourceX < bounds.Max.X; sourceX++ {
				targetX := bounds.Max.Y - sourceY - 1
				targetY := sourceX - bounds.Min.X
				rotated.Set(targetX, targetY, source.At(sourceX, sourceY))
			}
		}
		sidewaysCardBack = rotated
	})
	if sidewaysCardBack == nil {
		return cardBackImage()
	}
	result := canvas.NewImageFromImage(sidewaysCardBack)
	result.FillMode = canvas.ImageFillContain
	result.ScaleMode = canvas.ImageScaleSmooth
	return result
}

func (tile *CardTile) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(tile.image, tile.selectionBorder))
}

func (tile *CardTile) MinSize() fyne.Size {
	return tile.size
}

func (tile *CardTile) Tapped(_ *fyne.PointEvent) {
	tile.preview()
	if tile.OnActivate != nil {
		tile.OnActivate()
	}
}

func (tile *CardTile) MouseIn(_ *desktop.MouseEvent) {
	tile.preview()
}

func (tile *CardTile) MouseMoved(_ *desktop.MouseEvent) {
}

func (tile *CardTile) MouseOut() {
}

func (tile *CardTile) preview() {
	if tile.Card.ID != "" && tile.View.CardID != "" {
		if tile.OnPreview != nil {
			tile.OnPreview(tile.Card)
		}
		return
	}
	if tile.OnHiddenPreview != nil {
		tile.OnHiddenPreview()
	}
}
