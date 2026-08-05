# Simulator rules authority

This document records how Caster's Compendium determines rules behavior for
the simulator. It is an engineering policy, not an independent publication of
official tournament rules.

## Source precedence

When sources conflict, use the first applicable source in this list:

1. Current card text and current errata.
2. Current publisher rulings and FAQ entries.
3. The current Student Handbook.
4. Direct clarifications received from Speedrobo Games.
5. The retired Comprehensive Rules version 1.2 where they do not conflict with
   a newer rule.
6. A documented provisional simulator ruling when no published or clarified
   answer exists.

Speedrobo Games has confirmed to the maintainer that Comprehensive Rules
version 1.2 remain more or less accurate, with newer rules superseding older
rules where applicable.

## Primary references

- [Current Full Rule Book](https://speedrobogames.com/wp-content/uploads/2026/06/Full-Rule-Book.pdf)
- [Current FAQ](https://speedrobogames.com/games/the-caster-chronicles/the-caster-chronicles-faq/)
- [Comprehensive Rules version 1.2](https://speedrobogames.com/wp-content/uploads/2026/04/TCC_CR_1_2_EN.pdf)
- [Official errata](https://speedrobogames.com/games/the-caster-chronicles/kickstarter-errata/)

## Traceability requirements

Every implemented rule should have a stable rule key. Tests, implementation
comments where useful, and publisher questions should refer to that key rather
than relying only on prose or memory.

Use one of these statuses:

- `confirmed-modern`: explicitly supported by a current source.
- `confirmed-publisher`: supplied or approved directly by the publisher.
- `inherited-cr1.2`: taken from CR 1.2 and not contradicted by a newer source.
- `provisional`: a simulator interpretation awaiting confirmation.
- `superseded`: retained in the record but not used by the engine.

Suggested record shape:

| Key | Behavior | Authority | Status | Tests | Notes |
| --- | --- | --- | --- | --- | --- |
| `turn.phase-order` | Recovery, Draw, Call, Main, Battle, End | Handbook p.14 | `confirmed-modern` | Not started | |
| `turn.recovery` | At the start of a player's turn, all of that player's Rested cards become Recovered; Reversed cards remain Reversed | Handbook pp.6, 14 | `confirmed-modern` | `TestRecoverPlayerCardsRecoversOnlySpecifiedPlayersRestedFieldCards`; `TestCompleteEndPhaseRecoversIncomingPlayerAtomically` | Recovery completes before its priority sequence |
| `aether.production` | Resting a Caster produces Aether equal to its Level and of its Element; this action may be performed during either player's turn | Handbook pp.10, 16 | `confirmed-modern` | `TestValidateGenerateNonElementalAetherAllowsEitherPlayerWithoutMutation`; `TestGenerateNonElementalAetherRestsCasterAndUpdatesPool` | A Caster whose ability rests it performs that ability instead |
| `aether.payment` | Playing a non-Caster card costs its printed amount and requires at least one Aether matching that card's Element | Handbook p.10 | `confirmed-modern` | Not started | |
| `aether.non-elemental` | The starting Caster Token and a face-down Level 1 Caster produce non-elemental Aether | Handbook pp.13, 15 | `confirmed-modern` | `TestGenerateNonElementalAetherRestsCasterAndUpdatesPool`; `TestUseCasterTokenRemovesTokenAndProducesNonElementalAether`; `TestPlayerSessionGenerateNonElementalAetherAllowsNonActivePlayer`; `TestPlayerSessionUseCasterTokenUpdatesBothPublicViews` | The used token ceases to exist rather than entering Exile; Void remains a distinct Element |
| `aether.expiration` | All produced and unspent Aether is erased during the End phase | Handbook p.14 | `confirmed-modern` | `TestCompleteCurrentPhaseRunsRemainingSkeletonAndRollsTurn`; `TestCompleteEndPhaseRejectsMissingActivePlayerWithoutClearingAether` | Apply before turn handoff |
| `chase.lifo` | The latest Chase object resolves first | Handbook p.19; CR 1.2 §6.5 | `confirmed-modern` | Not started | |
| `chase.priority-after-link` | Turn player gains priority after one link resolves | CR 1.2 §6.5.1.b | `inherited-cr1.2` | Not started | |
| `caster.facedown-aether` | A face-down Caster produces non-elemental Aether | Handbook p.15 | `confirmed-modern` | Not started | Supersedes old Void behavior |

## Resolving uncertainty

When an ambiguity is found:

1. Stop that particular rule from silently becoming assumed behavior.
2. Search the current handbook, FAQ, errata, and applicable card text.
3. Compare the CR 1.2 rule.
4. Record the question and the best provisional interpretation.
5. Ask Speedrobo Games when the answer could affect legal plays.
6. Update the rule record and regression tests when confirmation arrives.

Publisher clarifications should record the date, question, answer, and the
maintainer's source for the conversation. Do not include private correspondence
or personal information in the public repository without permission.

## Known modern overrides

At minimum, do not inherit the following older behavior:

- Caster Tokens replace the former starting coin behavior.
- Void is an element; face-down Casters produce non-elemental Aether.
- Caster uniqueness considers name, subname, and traits.
- A level-up Caster must share a name with the Caster below it.
- A face-down Caster cannot be used as the lower card of a level-up.
- Current deck construction, formats, card text, and errata replace retired
  equivalents.
