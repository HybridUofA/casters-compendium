# First simulator vertical slice

This is the first functional implementation exercise for Hybrid. It deliberately
ends before card effects, combat, the Chase, GUI work, persistence, or
networking.

## Outcome

Starting from two valid deck lists and an injected deterministic random source,
the engine reaches the first player's Call Phase with a correct initial match
state and a reproducible event history.

## Required behavior

1. Create a unique match instance for every main-deck card copy.
2. Preserve each card's catalog definition ID separately from its match ID.
3. Assign immutable ownership.
4. Deterministically shuffle both decks.
5. Randomly choose the first player using the same injected random source.
6. Each player draws seven cards.
7. Each player places the next seven cards face down into an ordered Orb zone.
8. Each player independently keeps their hand or performs the one-time
   replacement:
   - choose any number of cards from hand;
   - place them on the bottom of the deck;
   - draw the same number;
   - do not reshuffle.
9. Place one Caster Token in each player's Caster Zone.
10. Enter the first turn's Recovery Phase.
11. Recover applicable cards.
12. Complete the Recovery priority point only through a temporary explicit
    placeholder decision; do not invent the full Chase implementation here.
13. Skip the first player's first Draw Phase draw.
14. Enter the first player's Call Phase.

## Suggested types to design

Design these on paper before committing to Go definitions:

- `PlayerID`
- `MatchCardID`
- `CardInstance`
- `Zone`
- `OrderedZone`
- `PlayerState`
- `MatchState`
- `Phase`
- `MatchStatus`
- an injected random-source interface
- setup commands and setup events

Questions each type should answer:

- Which fields are immutable after setup?
- Which collections have rules-significant order?
- Which data are private to one player?
- Can an invalid zero value exist?
- Is identity stable if a card changes zones or controllers?
- Can state be copied safely without aliasing mutable slices or maps?

## Command boundary

Setup should pause only where a player must decide something. Candidate setup
commands are:

- submit or confirm a deck;
- choose opening-hand replacement cards;
- keep the opening hand.

Shuffling, drawing, Orb placement, token placement, and automatic phase
advancement are engine consequences rather than separate UI commands.

## Initial invariants

Write tests for invariants before expanding the rules:

- Every physical card exists in exactly one place.
- A card's owner never changes.
- Match card IDs are unique.
- Each player begins with seven cards in hand before replacement.
- Each player has seven ordered, face-down Orbs.
- Neither player can inspect any Orb identity through their player view.
- Replacement preserves hand size and total card count.
- Replacement happens at most once per player.
- Replaced cards are placed on the bottom and the deck is not shuffled.
- Both players have exactly one initial Caster Token.
- The first player does not draw on their first turn.
- The same seed and choices produce identical state and events.
- Different player views do not expose the opponent's hand.

## Explicit non-goals

- Paying Aether
- Calling or leveling Casters
- Playing Servants, Barriers, or Conjures
- Card text and keywords
- Chase and general priority implementation
- Attacks and Orb corruption
- GUI rendering
- Network transport
- Replay-file compatibility

## Completion definition

The slice is complete when:

- its rules are recorded in the traceability table;
- its state transitions are implemented by Hybrid;
- unit tests cover successful setup, invalid decisions, invariants, hidden
  information, and determinism;
- `go test ./...` and the migrated-Fyne test configuration pass;
- a short design review confirms that no UI, HTTP, filesystem, or global
  randomness dependency entered the functional engine.

The next slice should add the phase skeleton and basic Call Phase behavior,
not the entire card-effect system.
