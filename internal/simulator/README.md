# Simulator packages

Simulator-specific application, session, and UI packages will live below this
directory. Shared card, deck, storage, and source packages remain under the
repository-level `internal` directory.

The simulator's functional game and networking code is intended to be authored
by Hybrid. See [`../../docs/simulator/architecture.md`](../../docs/simulator/architecture.md)
for the planned package boundaries and
[`../../docs/simulator/first-vertical-slice.md`](../../docs/simulator/first-vertical-slice.md)
for the first implementation exercise.

Rules are interpreted using the source precedence and traceability policy in
[`../../docs/simulator/rules-authority.md`](../../docs/simulator/rules-authority.md).

## v0.3.2 prototype boundary

The rc.4 prototype supports manual movement of controlled printed cards between
the selectable Hand, Deck, Graveyard, Exile, Caster, and Servant zones. Deck
moves choose the top or bottom explicitly, and field-to-field control transfers
preserve the card's face and orientation.

The following manual-effect cases remain deliberately deferred: moving tokens,
moving cards with Stock, inserting cards into Orbs, selecting hidden Deck or Orb
cards, and moving cards into or out of face-down Exile. These limitations keep
hidden-information grants, nested-card ownership, and ordered Orb behavior out
of the prototype until their rules are designed.

Planned casts may select recovered Caster-zone cards as Aether sources. Source
generation, payment, card play, Chase creation, priority transfer, and the
revision increment are accepted or rejected as one command.

## Deferred turn-flow issue

The prototype currently presents the End-to-Recovery transition as an action
for the outgoing active player. Revisit this after the main-game skeleton is in
place: determine which Recovery steps are automatic, perform those steps as
part of the turn handoff where appropriate, and ensure the incoming player is
the only player prompted for their next interactive decision.
