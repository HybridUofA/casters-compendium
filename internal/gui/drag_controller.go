package deckgui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
)

type CardDropTarget struct {
	Zone  decks.Zone
	Index int
}

type CardDragController struct {
	Layer *fyne.Container

	MainPanel fyne.CanvasObject
	SidePanel fyne.CanvasObject

	MainGrid *fyne.Container
	SideGrid *fyne.Container

	OnFinished func(
		source CardDragSource,
		target *CardDropTarget,
	)

	active bool

	source CardDragSource

	ghost       *canvas.Image
	placeholder *canvas.Rectangle

	target *CardDropTarget
}

// NewCardDragController coordinates drag ghosts, placeholders, and zone drop callbacks.
func NewCardDragController(
	layer *fyne.Container,
	mainPanel fyne.CanvasObject,
	sidePanel fyne.CanvasObject,
	mainGrid *fyne.Container,
	sideGrid *fyne.Container,
	onFinished func(
		CardDragSource,
		*CardDropTarget,
	),
) *CardDragController {
	return &CardDragController{
		Layer:      layer,
		MainPanel:  mainPanel,
		SidePanel:  sidePanel,
		MainGrid:   mainGrid,
		SideGrid:   sideGrid,
		OnFinished: onFinished,
	}
}

// Start initializes drag state and creates the floating visual representation.
func (controller *CardDragController) Start(
	tile *CardTile,
	source CardDragSource,
	position fyne.Position,
) {
	if controller.active {
		controller.cancel()
	}

	controller.active = true
	controller.source = source

	controller.ghost = controller.createGhost(source)

	ghostSize := tile.Size()

	if ghostSize.Width > 120 {
		ratio :=
			ghostSize.Height / ghostSize.Width

		ghostSize = fyne.NewSize(120, 120*ratio)
	}

	controller.ghost.Resize(ghostSize)
	controller.Layer.Add(controller.ghost)

	controller.placeholder =
		canvas.NewRectangle(
			color.NRGBA{
				R: 100,
				G: 160,
				B: 255,
				A: 70,
			},
		)

	if source.Kind == DragFromDeck {
		sourceGrid := controller.gridForZone(source.Zone)
		if sourceGrid != nil {
			sourceGrid.Objects = removeCanvasObject(sourceGrid.Objects, tile)
			relayout(sourceGrid)
			controller.placePlaceholder(source.Zone, source.Index)
		}
	}

	controller.moveGhost(position)
}

// Move updates the ghost and placeholder for the zone currently under the pointer.
func (controller *CardDragController) Move(
	_ *CardTile,
	_ CardDragSource,
	position fyne.Position,
) {
	if !controller.active {
		return
	}

	controller.moveGhost(position)
	zone, found := controller.zoneAt(position)
	if !found {
		controller.clearPlaceholder()
		return
	}

	controller.clearPlaceholder()

	grid := controller.gridForZone(zone)
	if grid == nil {
		return
	}

	index := insertionIndex(
		grid,
		position,
	)

	controller.placePlaceholder(
		zone,
		index,
	)
}

// End resolves the final drop target, invokes the callback, and clears drag visuals.
func (controller *CardDragController) End(
	_ *CardTile,
	_ CardDragSource,
	_ fyne.Position,
) {
	if !controller.active {
		return
	}
	source := controller.source
	target := controller.target
	controller.removeGhost()
	controller.clearPlaceholder()
	controller.active = false
	controller.target = nil
	if controller.OnFinished != nil {
		controller.OnFinished(source, target)
	}
}

// cancel clears all active drag state without producing a drop.
func (controller *CardDragController) cancel() {
	controller.removeGhost()
	controller.clearPlaceholder()
	controller.active = false
	controller.target = nil
}

// createGhost adds a translucent copy of the dragged tile to the overlay layer.
func (controller *CardDragController) createGhost(
	source CardDragSource,
) *canvas.Image {
	if path, found := cardimages.FindThumbnail(source.Card.ID); found {
		image := canvas.NewImageFromFile(path)
		image.FillMode = canvas.ImageFillContain
		image.ScaleMode = canvas.ImageScaleSmooth
		image.Translucency = 0.15

		return image
	}

	if path, found := cardimages.Find(source.Card.ID); found {
		image := canvas.NewImageFromFile(path)
		image.FillMode = canvas.ImageFillContain
		image.ScaleMode = canvas.ImageScaleSmooth
		image.Translucency = 0.15

		return image
	}

	image := canvas.NewImageFromResource(theme.BrokenImageIcon())
	image.FillMode = canvas.ImageFillContain

	return image
}

// moveGhost centers the floating drag image under the current pointer.
func (controller *CardDragController) moveGhost(
	absolutePosition fyne.Position,
) {
	if controller.ghost == nil {
		return
	}

	layerOrigin := fyne.CurrentApp().Driver().AbsolutePositionForObject(controller.Layer)
	size := controller.ghost.Size()

	controller.ghost.Move(
		fyne.NewPos(
			absolutePosition.X-layerOrigin.X-size.Width/2,
			absolutePosition.Y-layerOrigin.Y-size.Height/2,
		),
	)
	controller.ghost.Refresh()
}

// placePlaceholder shows the insertion position in the prospective destination grid.
func (controller *CardDragController) placePlaceholder(
	zone decks.Zone,
	index int,
) {
	grid := controller.gridForZone(zone)
	if grid == nil || controller.placeholder == nil {
		return
	}

	if index < 0 {
		index = 0
	}

	if index > len(grid.Objects) {
		index = len(grid.Objects)
	}

	grid.Objects = insertCanvasObject(grid.Objects, index, controller.placeholder)

	controller.target = &CardDropTarget{
		Zone:  zone,
		Index: index,
	}

	relayout(grid)
}

// clearPlaceholder removes the insertion marker and restores the affected grid.
func (controller *CardDragController) clearPlaceholder() {
	if controller.placeholder == nil {
		controller.target = nil
		return
	}

	for _, grid := range []*fyne.Container{
		controller.MainGrid,
		controller.SideGrid,
	} {
		if grid == nil {
			continue
		}

		updated := removeCanvasObject(
			grid.Objects,
			controller.placeholder,
		)

		if len(updated) != len(grid.Objects) {
			grid.Objects = updated
			relayout(grid)
		}
	}
	controller.target = nil
}

// removeGhost removes the floating drag image from the overlay.
func (controller *CardDragController) removeGhost() {
	if controller.ghost == nil || controller.Layer == nil {
		return
	}

	controller.Layer.Objects = removeCanvasObject(controller.Layer.Objects, controller.ghost)
}

// zoneAt identifies the deck zone containing an absolute pointer position.
func (controller *CardDragController) zoneAt(
	position fyne.Position,
) (decks.Zone, bool) {
	if containsAbsolutePosition(controller.MainPanel, position) {
		return decks.MainZone, true
	}

	if containsAbsolutePosition(controller.SidePanel, position) {
		return decks.SideZone, true
	}

	return decks.MainZone, false
}

// gridForZone returns the card grid associated with a deck zone.
func (controller *CardDragController) gridForZone(zone decks.Zone) *fyne.Container {
	switch zone {
	case decks.MainZone:
		return controller.MainGrid
	case decks.SideZone:
		return controller.SideGrid
	default:
		return nil
	}
}

// containsAbsolutePosition reports whether an absolute point lies inside an object.
func containsAbsolutePosition(object fyne.CanvasObject, position fyne.Position) bool {
	if object == nil {
		return false
	}

	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(object)

	size := object.Size()

	return position.X >= origin.X && position.X <= origin.X+size.Width && position.Y >= origin.Y && position.Y <= origin.Y+size.Height
}

// insertionIndex chooses the nearest before-or-after position for a pointer within a grid.
func insertionIndex(grid *fyne.Container, absolutePosition fyne.Position) int {
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(grid)
	localX := absolutePosition.X - origin.X
	localY := absolutePosition.Y - origin.Y

	for index, object := range grid.Objects {
		position := object.Position()
		size := object.Size()

		middleX := position.X + size.Width/2

		bottomY := position.Y + size.Height

		if localY <= bottomY && localX <= middleX {
			return index
		}
	}

	return len(grid.Objects)
}

// insertCanvasObject inserts an object at a clamped container index.
func insertCanvasObject(
	objects []fyne.CanvasObject,
	index int,
	object fyne.CanvasObject,
) []fyne.CanvasObject {
	if index < 0 {
		index = 0
	}
	if index > len(objects) {
		index = len(objects)
	}
	objects = append(objects, nil)
	copy(objects[index+1:], objects[index:])
	objects[index] = object
	return objects
}

// removeCanvasObject removes the first matching object from a container.
func removeCanvasObject(
	objects []fyne.CanvasObject,
	target fyne.CanvasObject,
) []fyne.CanvasObject {
	for index, object := range objects {
		if object != target {
			continue
		}
		return append(objects[:index], objects[index+1:]...)
	}
	return objects
}

// relayout reapplies a container's layout after direct object-slice changes.
func relayout(grid *fyne.Container) {
	if grid == nil {
		return
	}
	if grid.Layout != nil {
		grid.Layout.Layout(grid.Objects, grid.Size())
	}
	grid.Refresh()
}
