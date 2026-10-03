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
| `turn.phase-order` | Recovery, Draw, Call, Main, Battle, End | Handbook p.14 | `confirmed-modern` | `TestCompleteCurrentPhaseRunsRemainingSkeletonAndRollsTurn`; `TestPlayerSessionsRunPhaseSkeletonAndRollTurn`; `TestCompleteCurrentPhaseRejectsOpenOrInconsistentClosedPriorityWithoutMutation` | Each phase transition requires a completed priority sequence and opens a new sequence for the resulting phase |
| `turn.recovery` | At the start of a player's turn, all of that player's Rested cards become Recovered; Reversed cards remain Reversed | Handbook pp.6, 14 | `confirmed-modern` | `TestRecoverPlayerCardsRecoversOnlySpecifiedPlayersRestedFieldCards`; `TestCompleteEndPhaseRecoversIncomingPlayerAtomically` | Recovery completes before its priority sequence |
| `aether.production` | Resting a Caster produces Aether equal to its Level and of its Element; this action may be performed during either player's turn | Handbook pp.10, 16 | `confirmed-modern` | `TestCalculateCasterAetherMapsEveryElement`; `TestValidateGenerateCasterAetherAllowsControlledCasterWithoutMutation`; `TestGenerateCasterAetherUpdatesEveryElementalPool`; `TestPlayerSessionGenerateCasterAetherUpdatesSharedPublicViews` | A Caster whose ability rests it performs that ability instead |
| `aether.payment` | Playing a non-Caster card costs its printed amount and requires at least one Aether matching that card's Element | Handbook p.10 | `confirmed-modern` | Not started | |
| `aether.non-elemental` | The starting Caster Token and a face-down Level 1 Caster produce non-elemental Aether | Handbook pp.13, 15 | `confirmed-modern` | `TestGenerateNonElementalAetherRestsCasterAndUpdatesPool`; `TestUseCasterTokenRemovesTokenAndProducesNonElementalAether`; `TestPlayerSessionGenerateNonElementalAetherAllowsNonActivePlayer`; `TestPlayerSessionUseCasterTokenUpdatesBothPublicViews` | The used token ceases to exist rather than entering Exile; Void remains a distinct Element |
| `aether.expiration` | All produced and unspent Aether is erased during the End phase | Handbook p.14 | `confirmed-modern` | `TestCompleteCurrentPhaseRunsRemainingSkeletonAndRollsTurn`; `TestCompleteEndPhaseRejectsMissingActivePlayerWithoutClearingAether` | Apply before turn handoff |
| `chase.lifo` | The latest Chase object resolves first | Handbook p.19; CR 1.2 §6.5 | `confirmed-modern` | Not started | |
| `chase.priority-retained` | A player who takes an action other than passing retains priority | CR 1.2 §6.5.1.a | `inherited-cr1.2` | `TestCastServantPaysAndAddsCardToChase`; `TestAddChaseLinkAppendsLinkAndRetainsPriority`; `TestCastResolveAndClosePrioritySequence` | Confirm against a newer publisher source if one becomes available |
| `chase.pass-transfer` | A first pass transfers priority to the opponent | CR 1.2 §6.5.1.c | `inherited-cr1.2` | `TestPassPriorityRecordsFirstPassAndTransfersPriority` | |
| `chase.sequence-completion` | Consecutive passes complete the priority sequence when the Chase is empty | CR 1.2 §6.5.1.b | `inherited-cr1.2` | `TestPassPriorityClosesEmptySequenceOnSecondPass`; `TestPassPriorityRejectsClosedSequenceWithoutMutation`; `TestCastResolveAndClosePrioritySequence` | The engine represents closure with a false `PrioritySequenceOpen`, blank holder, zero passes, and an empty Chase |
| `chase.priority-after-link` | Turn player gains priority after one link resolves | CR 1.2 §6.5.1.b | `inherited-cr1.2` | `TestPassPriorityResolvesTopLinkOnSecondPass`; `TestResolveTopChaseLinkResolvesOnlyTopLinkAndResetsPriority`; `TestCastResolveAndClosePrioritySequence` | |
| `caster.facedown-aether` | A face-down Caster produces non-elemental Aether | Handbook p.15 | `confirmed-modern` | `TestValidateGenerateNonElementalAetherAllowsEitherPlayerWithoutMutation`; `TestGenerateNonElementalAetherRestsCasterAndUpdatesPool` | Supersedes old Void behavior |
| `match.loss.draw-phase-deck-out` | A player who cannot draw the required card during their Draw Phase loses the game | Handbook p.14; CR 1.2 §12.2.1 | `confirmed-modern` | `TestFinishMatchRecordsDecisiveResultAndClosesPriority`; `TestCompleteCurrentPhaseFinishesMatchOnDrawPhaseDeckOut` | This does not establish that every failed draw outside the Draw Phase causes a loss |
| `match.win.zero-orb-player-attack` | When a Servant attacks a player who already has no Orbs at battle calculation, the attacker's controller wins | Handbook p.18; CR 1.2 §8.4.5 | `confirmed-modern` | `TestPassPriorityWinsWhenPlayerAttackFindsZeroOrbs` | Removing the last Orb does not itself win; the FAQ confirms that Double Corrupt against one remaining Orb does not win |
| `match.simultaneous-loss` | If both players lose simultaneously, the game is a draw; otherwise the player who has not lost wins | CR 1.2 §§1.3.1–1.3.3 | `inherited-cr1.2` | `TestFinishMatchRecordsSimultaneousLossDraw` | No contradictory modern source found; no engine path produces this outcome yet |
| `battle.first-turn-skip` | The first player does not perform the Battle Phase on the first turn | Handbook p.14; CR 1.2 §5.6.1 | `confirmed-modern` | `TestCompleteCurrentPhaseRunsRemainingSkeletonAndRollsTurn`; `TestPlayerSessionsRunPhaseSkeletonAndRollTurn` | Main completion advances directly to End on turn 1 for the first player |
| `battle.mandatory-attacks` | The turn player cannot finish the Battle Phase while they control a Servant that is able to attack | Handbook pp.14, 17; current FAQ | `confirmed-modern` | Not started | “Able” requires an eligible attacker, legal target, and satisfiable attack requirements |
| `battle.attacker-eligibility` | A Recovered Servant controlled by the turn player can be selected to attack, including one played that turn unless an effect prohibits it | Handbook p.17 | `confirmed-modern` | `TestDeclareAttackRestsAttackerAndRecordsPlayerTarget`; `TestDeclareAttackRejectsInvalidRequestWithoutMutation` | Card restrictions such as Slow Start still apply |
| `battle.target-legality` | An attack may target an enemy Servant or the enemy player; while the defender controls a Reversed Servant, the enemy player is not a legal target, but other Servants remain legal targets | Handbook p.17; current FAQ | `confirmed-modern` | `TestDeclareAttackRecordsServantTarget`; `TestDeclareAttackRejectsInvalidRequestWithoutMutation` | |
| `battle.attack-declaration` | Declaring an attack selects attacker and target, Rests the attacker, pays any attack requirements, then opens a priority sequence before judgment | Handbook p.17; CR 1.2 §§8.3.3–8.3.6 | `confirmed-modern` | `TestDeclareAttackRestsAttackerAndRecordsPlayerTarget` | Declarer retains priority; judgment waits for the current priority sequence to complete |
| `battle.attack-interruption` | Before judgment, the attack stops if the attacker becomes Reversed or if the attacker or targeted Servant leaves the field | Current FAQ; Handbook p.21 | `confirmed-modern` | `TestPassPriorityClearsInterruptedAttackWithoutJudgment` | No judgment or Orb corruption occurs |
| `battle.attack-control-change` | Before judgment, the attack stops if control of the attacker changes | CR 1.2 §8.3.4.a | `inherited-cr1.2` | Not started | Confirm against a newer publisher source if one becomes available |
| `battle.servant-judgment` | Against a Recovered or Rested Servant compare attacker ATK to target ATK; against a Reversed Servant compare attacker ATK to target DEF; destroy the target only when attacker ATK is strictly higher | Handbook p.18; CR 1.2 §§8.4.3–8.4.4 | `confirmed-modern` | `TestPassPriorityResolvesServantJudgmentAndDestroysWeakerTarget`; `TestPassPriorityLeavesEqualServantAlive`; `TestPassPriorityUsesDefenseAgainstReversedServant` | Equal values do not destroy the target |
| `orb.corruption` | Corrupting an Orb moves the chosen enemy Orb to its owner's hand without revealing it to the opponent | Handbook p.18; current FAQ; CR 1.2 §10.9.1 | `confirmed-modern` | `TestPassPriorityCorruptsFrontmostOrbOnPlayerAttack` | First playable slice corrupts the frontmost Orb; explicit Orb choice remains deferred |
| `orb.multiple-corruption` | If multiple Orbs are corrupted simultaneously, move as many as exist to hand simultaneously; their owner chooses the order of optional Break plays | Current FAQ; CR 1.2 §§10.9.1.a, 11.3.3 | `confirmed-modern` | Not started | Double Corrupt against one Orb corrupts that Orb and does not win the game |
| `orb.break` | Break is optional when a corrupted Orb enters its owner's hand | Current FAQ | `confirmed-modern` | Not started | Break effect execution remains outside the first playable battle slice |

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
- A face-up Level 1 Caster conflicts only when another Caster has the same
  name, subname, and complete trait collection; sharing only one of those
  identity components is not sufficient.
- A level-up Caster must share a name with the Caster below it.
- A face-down Caster cannot be used as the lower card of a level-up.
- Current deck construction, formats, card text, and errata replace retired
  equivalents.
