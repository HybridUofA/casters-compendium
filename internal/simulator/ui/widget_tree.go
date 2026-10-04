package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func collectCardTiles(object fyne.CanvasObject) []*CardTile {
	if object == nil {
		return nil
	}
	if tile, ok := object.(*CardTile); ok {
		return []*CardTile{tile}
	}
	result := make([]*CardTile, 0)
	switch typed := object.(type) {
	case *fyne.Container:
		for _, child := range typed.Objects {
			result = append(result, collectCardTiles(child)...)
		}
	case *container.Scroll:
		result = append(result, collectCardTiles(typed.Content)...)
	case *zoneInteractLayer:
		result = append(result, collectCardTiles(typed.content)...)
	case *secondaryTapLayer:
		result = append(result, collectCardTiles(typed.content)...)
	}
	return result
}
