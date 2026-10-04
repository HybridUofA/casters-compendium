package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// secondaryTapLayer forwards right-clicks from empty/padding areas that are not
// themselves SecondaryTappable (for example deck count labels).
type secondaryTapLayer struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	onSecondary func(*fyne.PointEvent)
}

func newSecondaryTapLayer(content fyne.CanvasObject, onSecondary func(*fyne.PointEvent)) *secondaryTapLayer {
	layer := &secondaryTapLayer{content: content, onSecondary: onSecondary}
	layer.ExtendBaseWidget(layer)
	return layer
}

func (layer *secondaryTapLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(layer.content)
}

func (layer *secondaryTapLayer) TappedSecondary(event *fyne.PointEvent) {
	if layer.onSecondary != nil {
		layer.onSecondary(event)
	}
}
