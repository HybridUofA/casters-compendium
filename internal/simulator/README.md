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

## Deferred turn-flow issue

The prototype currently presents the End-to-Recovery transition as an action
for the outgoing active player. Revisit this after the main-game skeleton is in
place: determine which Recovery steps are automatic, perform those steps as
part of the turn handoff where appropriate, and ensure the incoming player is
the only player prompted for their next interactive decision.
