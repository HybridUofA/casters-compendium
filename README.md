# Caster's Compendium

Caster's Compendium is a desktop card browser and deck builder for Caster
Chronicles. It supports editable JSON decks, human-readable text decklists, and
Tabletop Simulator image sheets for both the main deck and sideboard.

## Features

### Deck building

- Build, load, rename, sort, and save complete main decks and sideboards while
  preserving each card's maximum copy limit.
- Drag cards from search results into either deck area, move them between the
  main deck and sideboard, and reorder them for personal theorycrafting and
  custom organization. **Sort Deck** restores the standard ordering.
- Preview card artwork on hover or click and read the card's full details in the
  information panel.
- Follow the operating-system theme or select a persistent light or dark theme.

### Search and card data

- Search the full card database by name, card type, trait, rules keyword,
  element, expansion, cost/level, and playtesting status.
- Extract keyword filters automatically from current card ability text instead
  of relying on a fixed keyword list.
- Download and cache the card database, artwork, and thumbnails locally for
  convenient reuse and faster subsequent launches.
- Download initial card assets concurrently and reuse a hash-verified GitHub
  snapshot when available.
- Compare a deterministic `cardlist.sha256` digest at startup, prompt when the
  published card list changes, or force a full database refresh from the main
  menu.

### Import, sharing, and export

- Import and save editable JSON deck files for sharing between players.
- Import and export human-readable text decklists containing card codes for
  tournament submission.
- Export both main-deck and sideboard image sheets for Tabletop Simulator using
  the bundled card-back asset.

### Desktop applications

- Download precompiled applications for Windows, macOS Intel, macOS Apple
  Silicon, and Linux x64.
- Install native packages on Debian and Ubuntu (`.deb`) or Arch Linux
  (`.pkg.tar.zst`).

## Roadmap

### Upcoming in v0.1.3

v0.1.3 is planned as a deckbuilder-focused refinement release before simulator
development becomes user-visible. Work targeted for this release includes:

- Working directly with Speedrobo on art assets and prototype cards.
- Showing card previews on hover instead of requiring a click.
- Allowing a card to be dragged from the main deck or sideboard back to the
  search area to remove it.
- Adding optional backgrounds, primarily eight element-themed designs and
  potentially five designs for the OC-tier Casters, subject to their creators'
  permission.

### Planned for v0.2.0

v0.2.0 is planned to introduce the first rudimentary simulator. Simulator work
is a large, experimental undertaking and remains a work in progress without a
promised delivery date.

## How to use the deck builder

1. Choose **Make a New Deck**, or choose **Load a Deck** to open an existing
   editable JSON deck or text decklist.
2. Find cards with the name, element, cost/level, type, trait, keyword,
   expansion, and playtesting filters in the Card Search panel. Keyword choices
   are extracted from the current card ability data, so newly published rules
   labels can appear without an application update.
3. Hover over a card to display its full image and card details. Clicking or
   tapping also works on devices without pointer hover.
4. Right-click a search result to add one copy to the Main Deck. Hold **Shift**
   while right-clicking to add one copy to the Side Deck. A search result can
   also be dragged directly into either deck area.
5. **Right-click a card already in the Main Deck or Side Deck to remove one
   copy.** Drag a deck card to reorder it or move it between deck areas.
6. Choose **Sort Deck** to restore the standard automatic ordering.

The deck controls provide the following file and export operations:

- **Save** and **Save As** write the editable JSON deck format.
- **Export Decklist** writes the human-readable text format used by
  `Arthur Test Deck.txt`.
- **Export Main** and **Export Sideboard** create Tabletop Simulator PNG sheets.
- **Rename** changes the deck's display and default export name.
- **Main Menu** returns to deck creation, file conversion, database update,
  appearance settings, and the built-in **How to Use** guide.

From the main menu, **Generate Deck Image from Decklist** creates a Tabletop
Simulator sheet without opening the deck editor, while **Generate Decklist
File** converts an editable JSON deck to the text interchange format.

## Running from source

Install Go and the native prerequisites listed in the
[Fyne quick-start documentation](https://docs.fyne.io/started/quick/), then run:

```sh
go run ./cmd/deckbuilder
```

Tests, including the stricter Fyne threading migration, run with:

```sh
go test ./...
go test -tags migrated_fynedo ./...
```

## Local application data

The applications share downloaded data under Fyne's per-user configuration
directory. The stable storage application ID remains
`io.github.hybriduofa.casterdeckbuilder`, even though the display name changed,
so existing deckbuilder installations continue to use the same files and the
future simulator will not create a second card database or image cache.

- Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/fyne/io.github.hybriduofa.casterdeckbuilder`
- macOS: `~/Library/Preferences/fyne/io.github.hybriduofa.casterdeckbuilder`
- Windows: `%APPDATA%\fyne\io.github.hybriduofa.casterdeckbuilder`

The directory contains `cards.json`, `cardlist.sha256`, `images/`, `thumbnails/`,
and the setup-completion marker. Deck and export files remain wherever the user
selected in the save dialog.

## Repository architecture

The repository is organized as one Go module with application-specific code
around shared card and game packages:

```text
cmd/
  deckbuilder/          Deckbuilder executable and Fyne metadata
  simulator/            Reserved simulator command
  tools/                Card-data and legacy CLI utilities
internal/
  carddata/             Catalog, local paths, image cache, and normalization
  deckbuilder/          Deckbuilder application, UI, and TTS export
  deckio/               Shared JSON and text deck formats
  game/                 Simulator-safe card and deck domain logic
  simulator/            Reserved simulator packages
  sources/              External card-data clients
data/                    Published card snapshot and bootstrap artwork
packaging/               Linux distribution and desktop package definitions
```

Packages under `internal/game` do not depend on Fyne, HTTP, or filesystem
storage. Both applications can consume the same game models, `deckio` formats,
and OS-local `carddata` paths without importing one another.

## Desktop packages and releases

The `Package desktop applications` GitHub Actions workflow builds native
artifacts for:

- Windows x64
- macOS Intel
- macOS Apple Silicon
- Linux x64
- Debian and Ubuntu x64 (`.deb`)
- Arch Linux x64 (`.pkg.tar.zst`)

A manual workflow run stores packages as build artifacts. Pushing a version tag
such as `v0.1.2` builds the same packages and publishes them as GitHub Release
assets. The macOS and Windows packages are currently unsigned; operating-system
security prompts may therefore require the user to explicitly allow the first
launch. Code signing can be added later when the appropriate Apple Developer and
Windows signing certificates are available.

Fyne's packaging tool uses `data/images/shadow.png` as the application icon and
embeds `MTD-back-ver01.png` separately for Tabletop Simulator exports.

Debian and Ubuntu users can install the native package with:

```sh
sudo apt install ./casters-compendium_0.1.2_amd64.deb
```

Arch Linux users can install the native package with:

```sh
sudo pacman -U casters-compendium-0.1.2-1-x86_64.pkg.tar.zst
```
