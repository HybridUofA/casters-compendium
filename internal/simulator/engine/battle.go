package engine

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func DeclareAttack(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	attackerID model.MatchCardID,
	targetKind model.AttackTargetKind,
	targetCardID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if strings.TrimSpace(string(attackerID)) == "" {
		return fmt.Errorf("attacker ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != model.PhaseBattle {
		return fmt.Errorf("must be in battle phase to declare an attack")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("must be active player to declare attacks")
	}
	if state.Attack.Step != model.BattleStepIdle || state.Attack.AttackerID != "" {
		return fmt.Errorf("an attack is already in progress")
	}
	if !state.PrioritySequenceOpen {
		return fmt.Errorf("priority sequence must be open")
	}
	if state.PriorityHolder != actingPlayerID {
		return fmt.Errorf("%q does not hold priority", actingPlayerID)
	}
	if state.PassCount != 0 {
		return fmt.Errorf("pass count must be 0")
	}
	if len(state.ChaseLinks) != 0 {
		return fmt.Errorf("chase must be empty to declare an attack")
	}

	actingIndex, opponentIndex, err := battlePlayerIndexes(state, actingPlayerID)
	if err != nil {
		return err
	}
	if err := validateAttacker(state, actingIndex, attackerID); err != nil {
		return err
	}
	if err := validateAttackTarget(state, catalog, attackerID, opponentIndex, targetKind, targetCardID); err != nil {
		return err
	}

	attacker := state.CardInstances[attackerID]
	attacker.Orientation = model.OrientationRested
	state.CardInstances[attackerID] = attacker
	state.Attack = model.AttackState{
		AttackerID:   attackerID,
		TargetKind:   targetKind,
		TargetCardID: targetCardID,
		Step:         model.BattleStepDeclared,
	}
	state.PassCount = 0
	state.PriorityHolder = actingPlayerID
	state.PrioritySequenceOpen = true
	state.Revision++
	return nil
}

func battlePlayerIndexes(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
) (actingIndex int, opponentIndex int, err error) {
	actingIndex = -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingIndex = index
			break
		}
	}
	if actingIndex == -1 {
		return -1, -1, fmt.Errorf("acting player ID not found")
	}
	return actingIndex, 1 - actingIndex, nil
}

func validateAttacker(
	state *model.MatchState,
	actingIndex int,
	attackerID model.MatchCardID,
) error {
	if !slices.Contains(state.Players[actingIndex].ServantZone, attackerID) {
		return fmt.Errorf("attacker %q is not in the acting player's servant zone", attackerID)
	}
	attacker, ok := state.CardInstances[attackerID]
	if !ok {
		return fmt.Errorf("attacker %q does not exist", attackerID)
	}
	if attacker.Controller != state.Players[actingIndex].ID {
		return fmt.Errorf("attacker %q is not controlled by the acting player", attackerID)
	}
	if attacker.Orientation != model.OrientationRecovered {
		return fmt.Errorf("attacker %q must be Recovered", attackerID)
	}
	return nil
}

func validateAttackTarget(
	state *model.MatchState,
	catalog rules.CardCatalog,
	attackerID model.MatchCardID,
	opponentIndex int,
	targetKind model.AttackTargetKind,
	targetCardID model.MatchCardID,
) error {
	opponent := state.Players[opponentIndex]
	switch targetKind {
	case model.AttackTargetPlayer:
		if targetCardID != "" {
			return fmt.Errorf("player attacks cannot name a target card")
		}
		if attackerHasPrintedHubris(state, catalog, attackerID) {
			return nil
		}
		for _, cardID := range opponent.ServantZone {
			instance, ok := state.CardInstances[cardID]
			if !ok {
				return fmt.Errorf("opponent servant %q does not exist", cardID)
			}
			if instance.Orientation == model.OrientationReversed {
				return fmt.Errorf("cannot attack the player while a reversed enemy servant is in play")
			}
		}
		return nil
	case model.AttackTargetServant:
		if strings.TrimSpace(string(targetCardID)) == "" {
			return fmt.Errorf("servant attacks require a target card")
		}
		if !slices.Contains(opponent.ServantZone, targetCardID) {
			return fmt.Errorf("target servant %q is not in the opponent's servant zone", targetCardID)
		}
		if _, ok := state.CardInstances[targetCardID]; !ok {
			return fmt.Errorf("target servant %q does not exist", targetCardID)
		}
		return nil
	default:
		return fmt.Errorf("invalid target kind %q", targetKind)
	}
}

func attackerHasPrintedHubris(
	state *model.MatchState,
	catalog rules.CardCatalog,
	attackerID model.MatchCardID,
) bool {
	if state == nil || catalog == nil {
		return false
	}
	instance, ok := state.CardInstances[attackerID]
	if !ok {
		return false
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return false
	}
	return abilityHasPrintedHubris(definition.Ability)
}

func abilityHasPrintedHubris(ability string) bool {
	normalized := strings.ToLower(ability)
	return strings.Contains(normalized, "[hubris]") || strings.Contains(normalized, "hubris")
}

// resolveDeclaredAttack applies battle judgment for the current declared attack.
// It clears Attack before returning. Callers own the revision increment.
func resolveDeclaredAttack(state *model.MatchState, catalog rules.CardCatalog) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Attack.Step != model.BattleStepDeclared {
		return fmt.Errorf("no declared attack to resolve")
	}

	attack := state.Attack
	activeIndex, opponentIndex, err := battlePlayerIndexes(state, state.Turn.ActivePlayer)
	if err != nil {
		return err
	}

	if attackInterrupted(state, activeIndex, opponentIndex, attack) {
		clearAttack(state)
		return nil
	}

	switch attack.TargetKind {
	case model.AttackTargetServant:
		if err := resolveServantJudgment(state, catalog, attack); err != nil {
			return err
		}
	case model.AttackTargetPlayer:
		awaitingOrb, err := resolvePlayerJudgment(state, catalog, opponentIndex, attack.AttackerID)
		if err != nil {
			return err
		}
		if awaitingOrb {
			return nil
		}
	default:
		return fmt.Errorf("invalid attack target kind %q", attack.TargetKind)
	}

	clearAttack(state)
	return nil
}

func attackInterrupted(
	state *model.MatchState,
	activeIndex int,
	opponentIndex int,
	attack model.AttackState,
) bool {
	attacker, ok := state.CardInstances[attack.AttackerID]
	if !ok {
		return true
	}
	if !slices.Contains(state.Players[activeIndex].ServantZone, attack.AttackerID) {
		return true
	}
	if attacker.Controller != state.Players[activeIndex].ID {
		return true
	}
	if attacker.Orientation == model.OrientationReversed {
		return true
	}
	if attack.TargetKind == model.AttackTargetServant {
		if !slices.Contains(state.Players[opponentIndex].ServantZone, attack.TargetCardID) {
			return true
		}
		if _, ok := state.CardInstances[attack.TargetCardID]; !ok {
			return true
		}
	}
	return false
}

func resolveServantJudgment(
	state *model.MatchState,
	catalog rules.CardCatalog,
	attack model.AttackState,
) error {
	attackerATK, err := cardCombatStat(state, catalog, attack.AttackerID, true)
	if err != nil {
		return fmt.Errorf("attacker combat value: %w", err)
	}
	target, ok := state.CardInstances[attack.TargetCardID]
	if !ok {
		return fmt.Errorf("target servant %q does not exist", attack.TargetCardID)
	}
	useDefense := target.Orientation == model.OrientationReversed
	targetValue, err := cardCombatStat(state, catalog, attack.TargetCardID, !useDefense)
	if err != nil {
		return fmt.Errorf("target combat value: %w", err)
	}
	if attackerATK > targetValue {
		if err := destroyServant(state, attack.TargetCardID); err != nil {
			return err
		}
	}
	return nil
}

// resolvePlayerJudgment finishes a zero-Orb win or pauses for Orb choice.
// When Orbs remain, Attack.Step becomes AwaitingJudgment and awaitingOrb is true.
func resolvePlayerJudgment(
	state *model.MatchState,
	catalog rules.CardCatalog,
	opponentIndex int,
	attackerID model.MatchCardID,
) (awaitingOrb bool, err error) {
	opponent := &state.Players[opponentIndex]
	if len(opponent.Orbs) == 0 {
		actingIndex := 1 - opponentIndex
		return false, finishMatch(state, model.MatchResult{
			Winner: state.Players[actingIndex].ID,
			Loser:  opponent.ID,
			Reason: model.EndReasonZeroOrbs,
		})
	}
	corruptCount := attackerCorruptCount(state, catalog, attackerID)
	if corruptCount > len(opponent.Orbs) {
		corruptCount = len(opponent.Orbs)
	}
	state.Attack.Step = model.BattleStepAwaitingJudgment
	state.Attack.CorruptCount = corruptCount
	return true, nil
}

func attackerCorruptCount(
	state *model.MatchState,
	catalog rules.CardCatalog,
	attackerID model.MatchCardID,
) int {
	if hasDoubleCorrupt(state, catalog, attackerID) {
		return 2
	}
	return 1
}

func hasDoubleCorrupt(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
) bool {
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return false
	}
	if instance.GrantedDoubleCorrupt {
		return true
	}
	if catalog == nil {
		return false
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return false
	}
	return cardDeclaresDoubleCorrupt(definition.Ability)
}

// CorruptOrb corrupts one enemy Orb. Prefer CorruptOrbs when Double Corrupt
// requires choosing multiple indexes at once.
func CorruptOrb(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	orbIndex int,
	expectedRevision model.Revision,
) error {
	return CorruptOrbs(state, catalog, actingPlayerID, []int{orbIndex}, expectedRevision)
}

// CorruptOrbs lets the active player choose which enemy Orbs to corrupt after a
// player-attack judgment. Indexes are 0-based into the defender's Orbs zone and
// must match Attack.CorruptCount exactly. Chosen Orbs move to hand simultaneously;
// Break offers are queued in selection order.
func CorruptOrbs(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	orbIndexes []int,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != model.PhaseBattle {
		return fmt.Errorf("must be in battle phase to corrupt an orb")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("must be active player to corrupt an orb")
	}
	if state.Attack.Step != model.BattleStepAwaitingJudgment {
		return fmt.Errorf("no player-attack judgment awaiting orb choice")
	}
	if state.Attack.TargetKind != model.AttackTargetPlayer {
		return fmt.Errorf("orb corruption requires a player-targeted attack")
	}
	if state.PrioritySequenceOpen || state.PriorityHolder != "" || state.PassCount != 0 {
		return fmt.Errorf("orb choice requires a closed priority sequence")
	}
	if len(state.ChaseLinks) != 0 {
		return fmt.Errorf("chase must be empty to corrupt an orb")
	}
	if state.PendingBreak.PlayerID != "" || len(state.PendingBreak.CardIDs) > 0 {
		return fmt.Errorf("a break decision is already pending")
	}
	wantCount := state.Attack.CorruptCount
	if wantCount < 1 {
		wantCount = 1
	}
	if len(orbIndexes) != wantCount {
		return fmt.Errorf("must choose exactly %d orb(s); got %d", wantCount, len(orbIndexes))
	}

	_, opponentIndex, err := battlePlayerIndexes(state, actingPlayerID)
	if err != nil {
		return err
	}
	opponent := &state.Players[opponentIndex]
	seen := make(map[int]struct{}, len(orbIndexes))
	for _, orbIndex := range orbIndexes {
		if orbIndex < 0 || orbIndex >= len(opponent.Orbs) {
			return fmt.Errorf("orb index %d is out of range for %d orbs", orbIndex, len(opponent.Orbs))
		}
		if _, dup := seen[orbIndex]; dup {
			return fmt.Errorf("orb index %d selected more than once", orbIndex)
		}
		seen[orbIndex] = struct{}{}
	}

	// Remove highest indexes first so lower indexes stay valid, but corrupt
	// into hand in the attacker's selection order.
	sortedDescending := append([]int(nil), orbIndexes...)
	slices.Sort(sortedDescending)
	slices.Reverse(sortedDescending)
	removedByIndex := make(map[int]model.MatchCardID, len(orbIndexes))
	for _, orbIndex := range sortedDescending {
		orbID := opponent.Orbs[orbIndex]
		opponent.Orbs = slices.Delete(opponent.Orbs, orbIndex, orbIndex+1)
		removedByIndex[orbIndex] = orbID
	}
	corrupted := make([]model.MatchCardID, 0, len(orbIndexes))
	for _, orbIndex := range orbIndexes {
		orbID := removedByIndex[orbIndex]
		opponent.Hand = append(opponent.Hand, orbID)
		if instance, ok := state.CardInstances[orbID]; ok {
			instance.Face = model.CardFaceDown
			state.CardInstances[orbID] = instance
		}
		model.MarkCardKnown(state, opponent.ID, orbID)
		corrupted = append(corrupted, orbID)
	}
	clearAttack(state)
	queueBreakIfPresent(state, catalog, opponent.ID, corrupted...)
	if state.PendingBreak.PlayerID == "" {
		reopenPriorityForActivePlayer(state)
	}
	state.Revision++
	return nil
}

func destroyServant(state *model.MatchState, cardID model.MatchCardID) error {
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("destroyed servant %q does not exist", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate destroyed servant: %w", err)
	}
	if location.Zone != model.ZoneServant {
		return fmt.Errorf("destroyed servant %q is not in a servant zone", cardID)
	}
	zone := &state.Players[location.PlayerIndex].ServantZone
	*zone = slices.Delete(*zone, location.CardIndex, location.CardIndex+1)

	ownerIndex := -1
	for index, player := range state.Players {
		if player.ID == instance.Owner {
			ownerIndex = index
			break
		}
	}
	if ownerIndex == -1 {
		return fmt.Errorf("owner %q of destroyed servant was not found", instance.Owner)
	}
	instance.Controller = instance.Owner
	instance.Face = model.CardFaceUp
	instance.Orientation = model.OrientationRecovered
	instance.GrantedDoubleCorrupt = false
	state.Players[ownerIndex].Graveyard = append(state.Players[ownerIndex].Graveyard, cardID)
	state.CardInstances[cardID] = instance
	return nil
}

func cardCombatStat(
	state *model.MatchState,
	catalog rules.CardCatalog,
	cardID model.MatchCardID,
	useAttack bool,
) (int, error) {
	if catalog == nil {
		return 0, fmt.Errorf("card catalog is required for battle judgment")
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return 0, fmt.Errorf("card %q does not exist", cardID)
	}
	definition, found := catalog.FindByID(string(instance.CardID))
	if !found {
		return 0, fmt.Errorf("definition %q was not found", instance.CardID)
	}
	raw := definition.Defense
	label := "defense"
	if useAttack {
		raw = definition.Attack
		label = "attack"
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("card %q has invalid %s value %q", cardID, label, raw)
	}
	return value, nil
}

func clearAttack(state *model.MatchState) {
	state.Attack = model.AttackState{}
}

func reopenPriorityForActivePlayer(state *model.MatchState) {
	state.PrioritySequenceOpen = true
	state.PassCount = 0
	state.PriorityHolder = state.Turn.ActivePlayer
}
