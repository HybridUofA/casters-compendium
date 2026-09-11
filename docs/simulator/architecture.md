# Simulator architecture

The simulator is designed as a deterministic rules engine with presentation and
network adapters around it. GUI widgets must not directly mutate authoritative
match state.

## Dependency direction

```text
internal/game/cards      internal/game/decks
          \                   /
           \                 /
            simulator model
                  |
            simulator rules
                  |
            simulator engine
             /           \
      player views      event log
          /                 \
   local session       network session
          \                 /
             simulator UI
```

Dependencies flow downward toward shared domain types. The functional engine
must not import Fyne, HTTP, sockets, filesystem storage, or card-image code.

## Planned responsibilities

The names below are architectural destinations, not a requirement to create all
packages before they contain meaningful behavior.

### `internal/simulator/model`

Plain game-state vocabulary:

- match and player identity;
- individual card-instance identity;
- ownership and control;
- ordered and unordered zones;
- card orientation and position;
- phase, turn, priority, and consecutive-pass state;
- Aether pools;
- the Chase;
- attacks and pending player choices;
- continuous-effect durations and match results.

The catalog card ID identifies a printed card definition. A match card ID
identifies one physical copy for the lifetime of a match.

### `internal/simulator/rules`

Pure legality and calculation functions:

- legal commands for the current state;
- legal targets and choices;
- payment validation;
- Caster uniqueness and level-up validation;
- mandatory-attack detection;
- combat values;
- visibility decisions;
- victory and loss checks.

Rule checks inspect state and return answers. They do not perform UI work or
silently mutate state.

The intended player experience is EDOPro-style rules enforcement: players
should only be offered actions, timings, choices, and targets that are currently
legal. The rules layer should therefore support both:

- enumerating or querying legal actions for presentation; and
- authoritatively validating a submitted command before it changes state.

The second check remains necessary even when the UI hides or disables illegal
actions. Future network clients and stale interfaces must not be able to bypass
the engine by constructing a command directly.

### `internal/simulator/engine`

The authoritative transition coordinator:

1. receive a semantic player command;
2. verify player, turn, revision, timing, costs, and targets;
3. reject it without mutation or accept it atomically;
4. apply state transitions;
5. emit ordered domain events;
6. perform required rule processes;
7. collect triggers and pending choices;
8. return the new state and events.

The UI requests `CallCaster`, `DeclareAttack`, or `PassPriority`; it does not
request arbitrary movement between zones.

### `internal/simulator/effects`

Card-effect behavior:

- continuous, activated, automatic, one-time, and replacement effects;
- costs, modes, targets, and resolution;
- last-known information;
- duration and expiration;
- keyword behavior.

Do not start by building a general card-text parser. Implement typed effect
definitions for a deliberately small card pool and generalize only after
repeated patterns are visible.

### `internal/simulator/view`

Projects authoritative state into information a particular viewer may know.
It must conceal hands, face-down Orbs, face-down Casters, and any other private
information while exposing public counts and public zones.

Future network clients receive a player view, never unrestricted match state.

### `internal/simulator/session`

Coordinates engine use:

- local two-player or pass-and-play sessions;
- command sequencing and state revisions;
- replay/event-log ownership;
- eventual authoritative host and client adapters.

Transport messages should carry semantic commands. Networking must not contain
a second implementation of game rules.

### `internal/simulator/ui`

Fyne presentation and input translation:

- battlefield rendering;
- display or enable only legal actions and targets supplied by the rules layer;
- Chase and priority presentation;
- pending-choice dialogs;
- animation and accessibility;
- conversion of user gestures into semantic commands.

The UI must not independently reproduce timing or legality rules. It asks the
rules/engine boundary what is legal and presents that answer, so a visual
control cannot drift away from authoritative behavior.

## Determinism

Given the same initial decks, rules/catalog version, random seed, and accepted
command sequence, the engine must produce the same events and resulting state.

Accordingly:

- inject randomness rather than reading global randomness;
- never depend on map iteration order for game decisions;
- represent simultaneous ordering as an explicit player choice;
- keep wall-clock time out of rules unless a match format explicitly uses it;
- attach a monotonically increasing revision to accepted state;
- record enough information to replay random results.

## Atomic commands and ordered events

A command represents intent. Events record accepted facts.

```text
Command: DeclareAttack(attacker, target)

Possible events:
  AttackDeclared
  CardRested
  PriorityGranted
```

Rejected commands produce an informative error and no partial state change.
Costs paid as part of an accepted play remain paid if the resulting Chase link
later resolves only partially.

## Rule-processing checkpoint

After relevant transitions and after each individual Chase link resolves, the
engine repeatedly:

1. applies immediate rules processes;
2. detects loss or draw conditions;
3. queues newly triggered abilities;
4. requests ordering when simultaneous triggers require a player choice;
5. places the next eligible trigger into the Chase;
6. grants priority when the state is stable.

Two consecutive passes resolve only the top Chase link. Players can receive
priority again before the next link resolves.

## Battle as a sub-state machine

Battle is not one indivisible operation:

```text
begin battle
  -> select mandatory eligible attacker
  -> select legal target
  -> pay attack requirements and rest attacker
  -> priority window
  -> verify attacker and target still qualify
  -> battle judgment or Orb corruption
  -> priority/trigger handling
  -> battle cleanup
  -> repeat while an eligible mandatory attacker exists
```

The active attack must be explicit state because removal, a control change, or
reversing the attacker can terminate it before judgment.

## Deferred decisions

Do not settle these through accidental implementation:

- serialization format for saved matches and replays;
- public network protocol compatibility;
- scripting language or data format for arbitrary card effects;
- automated handling of loops;
- clocks, spectators, reconnects, or matchmaking;
- visual layout and animation.

First establish a tested offline engine vocabulary and turn skeleton.

## Long-term legality requirement

A normal player interaction must not be capable of creating an illegal game
state. At each decision point, the application should:

1. determine which player, if any, may act;
2. determine the legal action kinds at the current timing;
3. determine legal sources, modes, costs, and targets for the selected action;
4. present only those choices;
5. validate the resulting command again;
6. apply an accepted command atomically or leave state unchanged.

This is a long-term engine and interaction requirement. It is intentionally not
part of the initial setup vertical slice.
