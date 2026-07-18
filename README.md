# Caster's Compendium

Caster's Compendium is a desktop card browser and deck builder for Caster
Chronicles. It supports editable JSON decks, human-readable text decklists, and
Tabletop Simulator image sheets for both the main deck and sideboard.

## Features

- Build, load, rename, sort, and save decks.
- Import and export the text format used by `Arthur Test Deck.txt`.
- Export main-deck and sideboard sheets for Tabletop Simulator.
- Install the required card back from the bundled application asset.
- Download card data, artwork, and thumbnails concurrently during first setup.
- Compare a deterministic `cardlist.sha256` digest at startup and prompt when the
  published card list changes.
- Force a full card-database refresh from the main menu.

## Running from source

Install Go and the native prerequisites listed in the
[Fyne quick-start documentation](https://docs.fyne.io/started/quick/), then run:

```sh
go run ./cmd/gui
```

Tests, including the stricter Fyne threading migration, run with:

```sh
go test ./...
go test -tags migrated_fynedo ./...
```

## Local application data

The application keeps downloaded data under Fyne's per-user configuration
directory. The stable application ID remains
`io.github.hybriduofa.casterdeckbuilder`, even though the display name changed,
so existing installations continue to use the same files.

- Linux: `${XDG_CONFIG_HOME:-$HOME/.config}/fyne/io.github.hybriduofa.casterdeckbuilder`
- macOS: `~/Library/Preferences/fyne/io.github.hybriduofa.casterdeckbuilder`
- Windows: `%APPDATA%\fyne\io.github.hybriduofa.casterdeckbuilder`

The directory contains `cards.json`, `cardlist.sha256`, `images/`, `thumbnails/`,
and the setup-completion marker. Deck and export files remain wherever the user
selected in the save dialog.

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
such as `v0.1.1` builds the same packages and publishes them as GitHub Release
assets. The macOS and Windows packages are currently unsigned; operating-system
security prompts may therefore require the user to explicitly allow the first
launch. Code signing can be added later when the appropriate Apple Developer and
Windows signing certificates are available.

Fyne's packaging tool uses `data/images/MTD-back-ver01.png` as the application
icon and embeds the card back separately for Tabletop Simulator exports.

Debian and Ubuntu users can install the native package with:

```sh
sudo apt install ./casters-compendium_0.1.1_amd64.deb
```

Arch Linux users can install the native package with:

```sh
sudo pacman -U casters-compendium-0.1.1-1-x86_64.pkg.tar.zst
```
