package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// zoneInteractLayer forwards primary/secondary taps from zone chrome that is
// not itself tappable (spacers, labels, empty pile area).
type zoneInteractLayer struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	onTap       func()
	onSecondary func(*fyne.PointEvent)
}

func newZoneInteractLayer(
	content fyne.CanvasObject,
	onTap func(),
	onSecondary func(*fyne.PointEvent),
) *zoneInteractLayer {
	layer := &zoneInteractLayer{content: content, onTap: onTap, onSecondary: onSecondary}
	layer.ExtendBaseWidget(layer)
	return layer
}

func (layer *zoneInteractLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(layer.content)
}

func (layer *zoneInteractLayer) Tapped(_ *fyne.PointEvent) {
	if layer.onTap != nil {
		layer.onTap()
	}
}

func (layer *zoneInteractLayer) TappedSecondary(event *fyne.PointEvent) {
	if layer.onSecondary != nil {
		layer.onSecondary(event)
	}
}
