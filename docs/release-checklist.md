# Release candidate checklist

Copy this checklist into the release pull request or tracking issue. Record the
candidate tag, source commit, tester, date, platform, and exact package tested.
Use `N/A` only with a short explanation.

## Candidate identity

- [ ] Candidate tag:
- [ ] Source commit:
- [ ] Tester and date:
- [ ] Packages tested:
- [ ] `VERSION`, package metadata, release notes, and website agree.

## Automated validation

- [ ] Continuous integration is green on the candidate commit.
- [ ] Vulnerability scanning is green.
- [ ] Every expected platform package built.
- [ ] SHA-256 checksums, SPDX SBOM, and provenance attestations exist.
- [ ] Downloaded package checksums verify successfully.

## Installation and startup

- [ ] A fresh installation reaches first-time setup.
- [ ] An existing installation retains its decks and settings.
- [ ] The main menu and deck editor open without errors.
- [ ] Diagnostic information reports the intended public version.

## Appearance

- [ ] The default appearance renders correctly.
- [ ] Light and dark themes render correctly.
- [ ] Every bundled background is visibly rendered.
- [ ] Appearance settings persist after restart.
- [ ] Text, cards, and controls remain readable over every background.

## Core deck behavior

- [ ] Create a deck and add, remove, reorder, and multi-select cards.
- [ ] Move selected cards between the main deck and side deck.
- [ ] Save and reload a deck without changing its contents.
- [ ] Import and export both native and Speedrobo-compatible text decklists.
- [ ] Invalid or malformed input produces an informative error.

## Tabletop Simulator

- [ ] Export a hosted TTS deck and load it from a separate network connection.
- [ ] Confirm card faces and the bundled card back load over HTTPS.
- [ ] Generate and use the local-export fallback.

## Platform acceptance

- [ ] Windows x64 package launches and completes the applicable checks.
- [ ] Debian/Ubuntu x64 package launches and completes the applicable checks.
- [ ] Arch Linux x64 package launches and completes the applicable checks.
- [ ] Generic Linux x64 archive launches and completes the applicable checks.
- [ ] macOS Intel package is tester-confirmed, or the limitation is recorded.
- [ ] macOS Apple Silicon package is tester-confirmed, or the limitation is recorded.

## Publication

- [ ] Known limitations and deferred issues are documented.
- [ ] Release notes describe user-visible changes and compatibility concerns.
- [ ] The stable tag points to the exact accepted source commit.
- [ ] GitHub release downloads and website links resolve.
- [ ] Discord announcement contains working release and download links.
- [ ] A short post-release launch and core-flow check passed.
