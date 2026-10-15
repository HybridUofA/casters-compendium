package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	dataassets "github.com/HybridUofA/casters-compendium/data"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// CardTile is the simulator's presentation-only card widget. Hover previews
// are desktop-friendly, while tapping provides equivalent touch behavior.
type CardTile struct {
	widget.BaseWidget

	View                simulatorview.CardView
	Card                cards.Card
	baseSize            fyne.Size
	size                fyne.Size
	image               *canvas.Image
	uprightImage        *canvas.Image
	sidewaysImage       *canvas.Image
	reversedImage       *canvas.Image
	selectionBorder     *canvas.Rectangle
	statusBadge         *canvas.Text
	statBackdrop        *canvas.Rectangle
	atkLabel            *canvas.Text
	defLabel            *canvas.Text
	selected            bool
	OnPreview           func(cards.Card)
	OnHiddenPreview     func()
	OnActivate          func()
	OnSecondaryActivate func(*fyne.PointEvent)
	OnDragBegin         func()
	OnDragUpdate        func(fyne.Position)
	OnDragEnd           func()
	dragStarted         bool
	cardBackURL         string
	sideways            bool
}

var _ desktop.Hoverable = (*CardTile)(nil)

var (
	sidewaysCardBackOnce  sync.Once
	sidewaysCardBack      image.Image
	cardImageCache        sync.Map
	rotatedCardImageCache sync.Map
)

func NewCardTile(
	cardView simulatorview.CardView,
	card cards.Card,
	size fyne.Size,
	onPreview func(cards.Card),
	onHiddenPreview func(),
) *CardTile {
	return newOrientedCardTile(cardView, card, size, onPreview, onHiddenPreview, false)
}

// newOrientedCardTile establishes the initial orientation before Fyne creates
// a renderer for the widget. Calling SetSideways while constructing a board
// would eagerly create and refresh a renderer that is immediately replaced as
// part of the board update.
func newOrientedCardTile(
	cardView simulatorview.CardView,
	card cards.Card,
	size fyne.Size,
	onPreview func(cards.Card),
	onHiddenPreview func(),
	sideways bool,
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
		tile.uprightImage = simulatorCardImage(card)
	} else {
		tile.uprightImage = cardBackImage()
	}
	// Keep one stable canvas.Image in the renderer tree. Orientation changes
	// mutate its contents instead of swapping the Objects() pointer (Fyne can
	// keep showing the first Objects() entry across Refresh).
	tile.image = cloneCanvasImageShell(tile.uprightImage)
	tile.selectionBorder = canvas.NewRectangle(color.Transparent)
	tile.selectionBorder.StrokeColor = theme.Color(theme.ColorNamePrimary)
	tile.selectionBorder.StrokeWidth = 4
	tile.selectionBorder.Hide()
	tile.statusBadge = canvas.NewText("DC", color.NRGBA{R: 220, G: 180, B: 60, A: 255})
	tile.statusBadge.TextStyle = fyne.TextStyle{Bold: true}
	tile.statusBadge.TextSize = 11
	tile.statusBadge.Hide()
	tile.statBackdrop = canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 170})
	tile.statBackdrop.Hide()
	tile.atkLabel = canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	tile.atkLabel.TextStyle = fyne.TextStyle{Bold: true}
	tile.atkLabel.TextSize = 10
	tile.atkLabel.Hide()
	tile.defLabel = canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	tile.defLabel.TextStyle = fyne.TextStyle{Bold: true}
	tile.defLabel.TextSize = 10
	tile.defLabel.Hide()
	// Apply orientation before ExtendBaseWidget so the first paint is correct.
	tile.setSidewaysState(sideways)
	tile.refreshStatusBadge()
	tile.refreshCombatStats()
	tile.ExtendBaseWidget(tile)
	return tile
}

func (tile *CardTile) refreshStatusBadge() {
	if tile == nil || tile.statusBadge == nil {
		return
	}
	if tile.View.Orientation == model.OrientationReversed {
		tile.statusBadge.Text = "REV"
		tile.statusBadge.Color = color.NRGBA{R: 220, G: 80, B: 80, A: 255}
		tile.statusBadge.Show()
		return
	}
	showDC := tile.View.GrantedDoubleCorrupt ||
		cardDeclaresDoubleCorruptAbility(tile.Card.Ability)
	if showDC && tile.View.MatchID != "" {
		tile.statusBadge.Text = "DC"
		tile.statusBadge.Color = color.NRGBA{R: 220, G: 180, B: 60, A: 255}
		tile.statusBadge.Show()
		return
	}
	tile.statusBadge.Hide()
}

var (
	statColorNeutral = color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	statColorBuff    = color.NRGBA{R: 80, G: 220, B: 120, A: 255}
	statColorNerf    = color.NRGBA{R: 240, G: 90, B: 90, A: 255}
)

// refreshCombatStats shows live ATK/DEF on face-up servants (MD/EDOPro-style).
func (tile *CardTile) refreshCombatStats() {
	if tile == nil || tile.atkLabel == nil || tile.defLabel == nil || tile.statBackdrop == nil {
		return
	}
	hide := func() {
		tile.atkLabel.Hide()
		tile.defLabel.Hide()
		tile.statBackdrop.Hide()
	}
	if !tile.View.ShowFace || tile.View.MatchID == "" {
		hide()
		return
	}
	printedATK, okATK := parsePrintedStat(tile.Card.Attack)
	printedDEF, okDEF := parsePrintedStat(tile.Card.Defense)
	if !okATK && !okDEF {
		hide()
		return
	}
	effATK := printedATK + tile.View.AttackModifier
	effDEF := printedDEF + tile.View.DefenseModifier
	tile.atkLabel.Text = fmt.Sprintf("%d", effATK)
	tile.defLabel.Text = fmt.Sprintf("%d", effDEF)
	tile.atkLabel.Color = combatStatColor(effATK, printedATK)
	tile.defLabel.Color = combatStatColor(effDEF, printedDEF)
	tile.atkLabel.Show()
	tile.defLabel.Show()
	tile.statBackdrop.Show()
}

func parsePrintedStat(raw string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return value, true
}

func combatStatColor(effective, printed int) color.Color {
	switch {
	case effective > printed:
		return statColorBuff
	case effective < printed:
		return statColorNerf
	default:
		return statColorNeutral
	}
}

// SetSelected displays whether this card is included in the pending UI choice.
func (tile *CardTile) SetSelected(selected bool) {
	if tile.selected == selected {
		return
	}
	tile.selected = selected
	if selected {
		tile.selectionBorder.Show()
	} else {
		tile.selectionBorder.Hide()
	}
	tile.selectionBorder.Refresh()
}

// SetSideways rotates card artwork clockwise. Portrait tiles swap their layout
// dimensions so a Rested card retains the same visual scale; already-landscape
// tiles such as Orbs retain their requested dimensions. This is presentation
// state only and does not change authoritative state.
func (tile *CardTile) SetSideways(sideways bool) {
	tile.setSidewaysState(sideways)
	tile.Refresh()
}

func (tile *CardTile) setSidewaysState(sideways bool) {
	tile.sideways = sideways
	tile.size = tile.baseSize
	if !sideways {
		if tile.View.Orientation == model.OrientationReversed {
			if tile.reversedImage == nil {
				if tile.View.ShowFace {
					tile.reversedImage = rotatedSimulatorCardImageByTurns(tile.Card, tile.uprightImage, 2)
				} else {
					tile.reversedImage = rotatedCanvasImage(rotatedCanvasImage(tile.uprightImage))
				}
			}
			adoptCanvasImageContents(tile.image, tile.reversedImage)
		} else {
			adoptCanvasImageContents(tile.image, tile.uprightImage)
		}
		return
	}
	if tile.sidewaysImage == nil {
		if tile.View.ShowFace {
			tile.sidewaysImage = rotatedSimulatorCardImage(tile.Card, tile.uprightImage)
		} else if tile.cardBackURL != "" {
			tile.sidewaysImage = sidewaysCardBackImageForURL(tile.cardBackURL)
		} else {
			tile.sidewaysImage = sidewaysCardBackImage()
		}
	}
	adoptCanvasImageContents(tile.image, tile.sidewaysImage)
	if tile.baseSize.Height > tile.baseSize.Width {
		tile.size = fyne.NewSize(tile.baseSize.Height, tile.baseSize.Width)
	}
}

func cloneCanvasImageShell(source *canvas.Image) *canvas.Image {
	if source == nil {
		return canvas.NewImageFromImage(nil)
	}
	cloned := canvas.NewImageFromImage(source.Image)
	cloned.Resource = source.Resource
	cloned.File = source.File
	cloned.FillMode = source.FillMode
	cloned.ScaleMode = source.ScaleMode
	cloned.Translucency = source.Translucency
	return cloned
}

func adoptCanvasImageContents(dst, src *canvas.Image) {
	if dst == nil || src == nil || dst == src {
		return
	}
	dst.Image = src.Image
	dst.Resource = src.Resource
	dst.File = src.File
	dst.FillMode = src.FillMode
	dst.ScaleMode = src.ScaleMode
	dst.Translucency = src.Translucency
}

func rotatedSimulatorCardImage(card cards.Card, fallback *canvas.Image) *canvas.Image {
	return rotatedSimulatorCardImageByTurns(card, fallback, 1)
}

func rotatedSimulatorCardImageByTurns(card cards.Card, fallback *canvas.Image, turns int) *canvas.Image {
	path, found := cardimages.FindThumbnail(card.ID)
	if !found {
		path, found = cardimages.Find(card.ID)
	}
	if !found {
		for range turns {
			fallback = rotatedCanvasImage(fallback)
		}
		return fallback
	}
	key := struct {
		path  string
		turns int
	}{path, turns}
	if cached, exists := rotatedCardImageCache.Load(key); exists {
		return newSimulatorCanvasImage(cached.(image.Image))
	}
	source, loaded := cachedCardImage(path)
	if !loaded {
		for range turns {
			fallback = rotatedCanvasImage(fallback)
		}
		return fallback
	}
	for range turns {
		source = rotateImageClockwise(source)
	}
	actual, _ := rotatedCardImageCache.LoadOrStore(key, source)
	return newSimulatorCanvasImage(actual.(image.Image))
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
	decoded, loaded := decodeCardImage(path)
	if !loaded {
		return nil, false
	}
	actual, _ := cardImageCache.LoadOrStore(path, decoded)
	return actual.(image.Image), true
}

func decodeCardImage(path string) (image.Image, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	if err != nil {
		return nil, false
	}
	return decoded, true
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
		sidewaysCardBack = rotateImageClockwise(source)
	})
	if sidewaysCardBack == nil {
		return cardBackImage()
	}
	result := canvas.NewImageFromImage(sidewaysCardBack)
	result.FillMode = canvas.ImageFillContain
	result.ScaleMode = canvas.ImageScaleSmooth
	return result
}

func rotatedCanvasImage(source *canvas.Image) *canvas.Image {
	if source == nil {
		return nil
	}
	bitmap := source.Image
	if bitmap == nil && source.Resource != nil {
		bitmap, _, _ = image.Decode(bytes.NewReader(source.Resource.Content()))
	}
	if bitmap == nil && source.File != "" {
		bitmap, _ = cachedCardImage(source.File)
	}
	if bitmap == nil {
		return source
	}
	result := canvas.NewImageFromImage(rotateImageClockwise(bitmap))
	result.FillMode = source.FillMode
	result.ScaleMode = source.ScaleMode
	return result
}

func rotateImageClockwise(source image.Image) image.Image {
	bounds := source.Bounds()
	rotated := image.NewNRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
	for sourceY := bounds.Min.Y; sourceY < bounds.Max.Y; sourceY++ {
		for sourceX := bounds.Min.X; sourceX < bounds.Max.X; sourceX++ {
			targetX := bounds.Max.Y - sourceY - 1
			targetY := sourceX - bounds.Min.X
			rotated.Set(targetX, targetY, source.At(sourceX, sourceY))
		}
	}
	return rotated
}

func (tile *CardTile) CreateRenderer() fyne.WidgetRenderer {
	return &cardTileRenderer{tile: tile}
}

type cardTileRenderer struct {
	tile *CardTile
}

func (r *cardTileRenderer) Layout(size fyne.Size) {
	r.tile.image.Move(fyne.NewPos(0, 0))
	r.tile.image.Resize(size)

	// Contained artwork can leave empty space around the card. Outline the
	// artwork, not that empty space, including when the card is rested.
	width, height := size.Width, size.Height
	if aspect := r.tile.image.Aspect(); aspect > 0 {
		width = min(width, height*aspect)
		height = width / aspect
	}
	border := r.tile.selectionBorder
	inset := min(border.StrokeWidth/2, min(width, height)/2)
	border.CornerRadius = min(width, height) * 0.06
	artX := (size.Width - width) / 2
	artY := (size.Height - height) / 2
	border.Move(fyne.NewPos(artX+inset, artY+inset))
	border.Resize(fyne.NewSize(max(0, width-2*inset), max(0, height-2*inset)))

	if badge := r.tile.statusBadge; badge != nil {
		badge.Resize(badge.MinSize())
		badge.Move(fyne.NewPos(artX+4, artY+2))
	}
	if r.tile.statBackdrop != nil && r.tile.atkLabel != nil && r.tile.defLabel != nil &&
		!r.tile.statBackdrop.Hidden {
		atkSize := r.tile.atkLabel.MinSize()
		defSize := r.tile.defLabel.MinSize()
		pad := float32(3)
		barH := max(atkSize.Height, defSize.Height) + pad*2
		barY := artY + height - barH - 2
		r.tile.statBackdrop.Move(fyne.NewPos(artX+2, barY))
		r.tile.statBackdrop.Resize(fyne.NewSize(max(0, width-4), barH))
		r.tile.atkLabel.Resize(atkSize)
		r.tile.defLabel.Resize(defSize)
		r.tile.atkLabel.Move(fyne.NewPos(artX+pad+2, barY+pad))
		r.tile.defLabel.Move(fyne.NewPos(artX+width-defSize.Width-pad-2, barY+pad))
	}
}

func (r *cardTileRenderer) MinSize() fyne.Size { return r.tile.MinSize() }

func (r *cardTileRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.tile.image,
		r.tile.selectionBorder,
		r.tile.statusBadge,
		r.tile.statBackdrop,
		r.tile.atkLabel,
		r.tile.defLabel,
	}
}

func (r *cardTileRenderer) Refresh() {
	r.tile.refreshStatusBadge()
	r.tile.refreshCombatStats()
	r.Layout(r.tile.Size())
	r.tile.image.Refresh()
	r.tile.selectionBorder.Refresh()
	if r.tile.statusBadge != nil {
		r.tile.statusBadge.Refresh()
	}
	if r.tile.statBackdrop != nil {
		r.tile.statBackdrop.Refresh()
	}
	if r.tile.atkLabel != nil {
		r.tile.atkLabel.Refresh()
	}
	if r.tile.defLabel != nil {
		r.tile.defLabel.Refresh()
	}
}

func (r *cardTileRenderer) Destroy() {}

func (tile *CardTile) MinSize() fyne.Size {
	return tile.size
}

func (tile *CardTile) Tapped(_ *fyne.PointEvent) {
	tile.preview()
	if tile.OnActivate != nil {
		tile.OnActivate()
	}
}

func (tile *CardTile) TappedSecondary(event *fyne.PointEvent) {
	if tile.OnSecondaryActivate != nil {
		tile.OnSecondaryActivate(event)
	}
}

func (tile *CardTile) Dragged(event *fyne.DragEvent) {
	if event == nil {
		return
	}
	if !tile.dragStarted {
		tile.dragStarted = true
		if tile.OnDragBegin != nil {
			tile.OnDragBegin()
		}
	}
	if tile.OnDragUpdate != nil {
		tile.OnDragUpdate(event.AbsolutePosition)
	}
}

func (tile *CardTile) DragEnd() {
	if !tile.dragStarted {
		return
	}
	tile.dragStarted = false
	if tile.OnDragEnd != nil {
		tile.OnDragEnd()
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
