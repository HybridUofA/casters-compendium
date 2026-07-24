# Contributing to Caster's Compendium

Thank you for helping improve Caster's Compendium. Contributions should remain
focused, testable, and respectful of the intellectual-property permissions that
make the project possible.

## Before starting

For bugs and features, check the roadmap and existing issues first. Open an
issue before undertaking a large user-interface, file-format, network, or game
rules change so the intended behavior can be agreed upon before implementation.

Do not add card artwork, game data, logos, or other third-party material without
documented permission from the relevant rights holder.

## Development setup

Install Go and the native prerequisites from the
[Fyne quick-start guide](https://docs.fyne.io/started/quick/). Run the
deckbuilder from the repository root with:

```sh
go run -tags migrated_fynedo ./cmd/deckbuilder
```

Before submitting a pull request, run:

```sh
gofmt -w .
go vet -tags migrated_fynedo ./...
go test ./...
go test -tags migrated_fynedo ./...
go build -tags migrated_fynedo ./cmd/deckbuilder
```

Continuous integration repeats these checks. CI checks formatting without
modifying a contributor's branch.

## Pull requests

Keep each pull request centered on one coherent change. Include:

- The problem and the chosen behavior.
- User-facing or compatibility effects.
- Automated tests for new logic and regressions.
- Manual validation for visual or platform-specific behavior.
- Documentation and release-note changes when appropriate.
- `Closes #123` when the pull request completes a tracked issue.

Preserve existing deck-file compatibility unless a deliberately versioned
migration has been designed and documented.

## Architecture

Core game rules under `internal/game` must remain independent of Fyne, HTTP, and
filesystem storage. Shared deck formats belong under `internal/deckio`, shared
card data under `internal/carddata`, and source-specific clients under
`internal/sources`.

Prefer small, behavior-preserving extractions over broad rewrites of the GUI.
Avoid moving domain rules into event handlers or duplicating validation in the
interface.

## AI assistance

Material AI assistance must be reviewed by the contributor and disclosed in a
way consistent with [AI_STATEMENT.md](AI_STATEMENT.md). Contributors remain
responsible for correctness, licensing, security, and the submitted result.

## Conduct and security

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Report
security problems privately according to [SECURITY.md](SECURITY.md).
