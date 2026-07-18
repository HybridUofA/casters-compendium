package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const howToUseMarkdown = `# How to Use Caster's Compendium

## Build a deck

1. Choose **Make a New Deck**, or load an existing JSON or text deck file.
2. Use the filters in **Card Search** to find cards. The **Keyword** filter is
   built automatically from the current card ability data, so it includes rules
   labels such as Break, Enter, Last Words, Quickcast, Unity, and others.
3. **Left-click** a search result to view its full image and details.
4. **Right-click** a search result to add one copy to the Main Deck.
5. Hold **Shift** while right-clicking to add one copy to the Side Deck.

You can also drag a search result directly into either deck area.

## Remove or move cards

- **Right-click a card already in the Main Deck or Side Deck to remove one copy.**
- Drag a deck card to reorder it or move it between the Main Deck and Side Deck.
- Choose **Sort Deck** to restore the standard automatic ordering.

## Save and export

- **Save** and **Save As** store an editable JSON deck.
- **Export Decklist** creates a human-readable text decklist.
- **Export Main** and **Export Sideboard** create Tabletop Simulator PNG sheets.
- The main menu can also create an image directly from a text decklist or convert a JSON deck into a text decklist.

## Card data and appearance

- **Update Card Database** checks for and installs current card data and artwork.
- **Settings** lets you follow the system theme or force Light or Dark mode.
`

// showHowToUseDialog displays the built-in feature and interaction guide.
func showHowToUseDialog(window fyne.Window) {
	guide := widget.NewRichTextFromMarkdown(howToUseMarkdown)
	guide.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(guide)
	scroll.SetMinSize(fyne.NewSize(700, 520))
	dialog.ShowCustom("How to Use", "Close", scroll, window)
}
