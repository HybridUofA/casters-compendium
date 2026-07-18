package deckgui

import "fyne.io/fyne/v2"

// CardGridLayout arranges cards from left to right and top to bottom.
//
// Each cell's height is calculated from its width, preserving the
// portrait card aspect ratio while the window resizes.
type CardGridLayout struct {
	Columns int

	// HeightToWidth is the card's height divided by its width.
	// For a 130x182 card:
	//
	//	182 / 130 = 1.4
	HeightToWidth float32

	Padding float32

	// Used only when Fyne calculates the container's minimum size.
	MinimumCellWidth float32
}

// Layout left-aligns portrait cards and fits them within the available width and height.
func (layout *CardGridLayout) Layout(
	objects []fyne.CanvasObject,
	size fyne.Size,
) {
	if layout.Columns <= 0 || len(objects) == 0 {
		return
	}

	padding := layout.Padding
	if padding < 0 {
		padding = 0
	}

	columnCount := float32(layout.Columns)

	totalHorizontalPadding :=
		padding * float32(layout.Columns-1)

	cellWidth :=
		(size.Width - totalHorizontalPadding) /
			columnCount

	// Protect against extremely small container sizes.
	if cellWidth < 1 {
		cellWidth = 1
	}

	heightToWidth := layout.HeightToWidth
	if heightToWidth <= 0 {
		heightToWidth = 1.4
	}

	cellHeight := cellWidth * heightToWidth

	rows := (len(objects) + layout.Columns - 1) / layout.Columns
	totalVerticalPadding := padding * float32(rows-1)
	availableHeight := size.Height - totalVerticalPadding
	if availableHeight > 0 {
		heightConstrainedWidth :=
			(availableHeight / float32(rows)) /
				heightToWidth
		if heightConstrainedWidth < cellWidth {
			cellWidth = heightConstrainedWidth
			cellHeight = cellWidth * heightToWidth
		}
	}
	if cellWidth < 1 {
		cellWidth = 1
		cellHeight = cellWidth * heightToWidth
	}

	for index, object := range objects {
		column := index % layout.Columns
		row := index / layout.Columns

		x := float32(column) *
			(cellWidth + padding)

		y := float32(row) *
			(cellHeight + padding)

		object.Move(fyne.NewPos(x, y))
		object.Resize(
			fyne.NewSize(
				cellWidth,
				cellHeight,
			),
		)
	}
}

// MinSize reports the grid footprint at its configured minimum cell width.
func (layout *CardGridLayout) MinSize(
	objects []fyne.CanvasObject,
) fyne.Size {
	if layout.Columns <= 0 || len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}

	padding := layout.Padding
	if padding < 0 {
		padding = 0
	}

	minimumWidth := layout.MinimumCellWidth
	if minimumWidth <= 0 {
		minimumWidth = 40
	}

	heightToWidth := layout.HeightToWidth
	if heightToWidth <= 0 {
		heightToWidth = 1.4
	}

	minimumHeight :=
		minimumWidth * heightToWidth

	usedColumns := layout.Columns
	if len(objects) < usedColumns {
		usedColumns = len(objects)
	}

	rows :=
		(len(objects) + layout.Columns - 1) /
			layout.Columns

	width :=
		float32(usedColumns)*minimumWidth +
			float32(usedColumns-1)*padding

	height :=
		float32(rows)*minimumHeight +
			float32(rows-1)*padding

	return fyne.NewSize(width, height)
}
