package deckbuilder

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
3. **Hover over** a card to view its full image and details. Clicking or tapping
   also works on devices without pointer hover.
4. **Right-click** a search result to add one copy to the Main Deck.
5. Hold **Shift** while right-clicking to add one copy to the Side Deck.

You can also drag a search result directly into either deck area.

## Remove or move cards

- **Right-click a card already in the Main Deck or Side Deck to remove one copy.**
- Drag a deck card onto the **Card Search** panel to remove one copy.
- Drag a deck card to reorder it or move it between the Main Deck and Side Deck.
- Hold **Control**, or **Command** on macOS, and click individual deck copies to
  select a batch. Release the key, then drag any selected copy to move the batch
  within or between deck areas.
- Choose **Sort Deck** to restore the standard automatic ordering.

## Save and export

- **Save** and **Save As** store an editable JSON deck.
- **Export Decklist** creates a Speedrobo-compatible text decklist.
- **Export Main** and **Export Sideboard** create Tabletop Simulator PNG sheets.
- **Install to TTS** installs a portable saved object using shared online card
  sheets and the MTD card back, so multiplayer participants can load its art.
  Standard TTS data locations are detected automatically; a custom location
  only needs to be selected once. Local sheets are used as an offline fallback.
- The main menu can also create an image directly from a text decklist or convert a JSON deck into a text decklist.

## Card data and appearance

- **Update Card Database** checks the publisher-authorized hosted catalog and
  installs a cryptographically verified card database and artwork.
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
