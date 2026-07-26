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
   Element choices use a compact four-column row, and the remaining filters are
   paired to leave more room for card thumbnails.
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

- **Save** and **Save As** store an editable JSON deck. Save dialogs begin in
  **Documents/Caster's Compendium/Decks**, which is created automatically.
- Use **Deck** below the primary editor controls to switch between JSON decks and
  compatible text decklists in that folder. **Refresh** rescans the folder, and
  **Open Folder** reveals it in the system file manager.
- Entries beginning with **Official** are bundled DD01–DD04 templates published
  by Speedrobo Games. They work offline and use **Save As** when you want to
  keep an edited personal copy.
- The last selected deck opens automatically on the next launch. Switching
  decks asks before discarding unsaved changes.
- The **Export** menu creates a Speedrobo-compatible text decklist or a
  Tabletop Simulator PNG sheet for the main deck or sideboard.
- Imported text decklists require a main deck. The side-deck heading and total
  are optional; omitting them creates an empty side deck.
- **Install to TTS** installs a portable saved object using shared online card
  sheets and the MTD card back, so multiplayer participants can load its art.
  Standard TTS data locations are detected automatically; a custom location
  only needs to be selected once. Local sheets are used as an offline fallback.
- The main menu can also create an image directly from a text decklist or convert a JSON deck into a text decklist.

## Card data and appearance

- The editor uses stable utility-panel sizes: card information stays on the
  left, search stays on the right, and the deck area receives remaining space.
  Constrained details and filter content scroll within their panels.
- **Update Card Database** checks the publisher-authorized hosted catalog and
  installs a cryptographically verified card database and artwork.
- **Diagnostic Information** displays a reviewable support summary that can be
  copied into a bug report. It excludes deck contents, credentials, usernames,
  and exact filesystem paths, and nothing is transmitted automatically.
- **Settings** lets you follow the system theme or force Light or Dark mode,
  and choose None, Academy Rift, or Caster Duel as the window background.
`

// showHowToUseDialog displays the built-in feature and interaction guide.
func showHowToUseDialog(window fyne.Window) {
	guide := widget.NewRichTextFromMarkdown(howToUseMarkdown)
	guide.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(guide)
	scroll.SetMinSize(fyne.NewSize(700, 520))
	dialog.ShowCustom("How to Use", "Close", scroll, window)
}
