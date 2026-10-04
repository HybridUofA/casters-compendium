package engine

import (
	"fmt"
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

type castValidator func(
	*model.MatchState,
	rules.CardCatalog,
	model.PlayerID,
	model.MatchCardID,
	model.AetherPayment,
) error

func CastServant(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, entryOrientation model.CardOrientation, expectedRevision model.Revision) error {
	validate := func(
		state *model.MatchState,
		catalog rules.CardCatalog,
		actingPlayerID model.PlayerID,
		cardID model.MatchCardID,
		payment model.AetherPayment,
	) error {
		return rules.ValidateCastServant(
			state,
			catalog,
			actingPlayerID,
			cardID,
			payment,
			entryOrientation,
		)
	}
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"servant",
		validate,
		entryOrientation,
	)
}

func CastServantWithPlan(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, plan model.CastPaymentPlan, entryOrientation model.CardOrientation, expectedRevision model.Revision) error {
	validate := func(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment) error {
		return rules.ValidateCastServant(state, catalog, actingPlayerID, cardID, payment, entryOrientation)
	}
	return castCardWithPlan(state, catalog, actingPlayerID, cardID, plan, expectedRevision, "servant", validate, entryOrientation)
}

func CastConjure(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) error {
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"conjure",
		rules.ValidateCastConjure,
		"",
	)
}

func CastConjureWithPlan(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) error {
	return castCardWithPlan(state, catalog, actingPlayerID, cardID, plan, expectedRevision, "conjure", rules.ValidateCastConjure, "")
}

func CastBarrier(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) error {
	return castCard(
		state,
		catalog,
		actingPlayerID,
		cardID,
		payment,
		expectedRevision,
		"barrier",
		rules.ValidateCastBarrier,
		"",
	)
}

func CastBarrierWithPlan(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) error {
	return castCardWithPlan(state, catalog, actingPlayerID, cardID, plan, expectedRevision, "barrier", rules.ValidateCastBarrier, "")
}

func castCard(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	payment model.AetherPayment,
	expectedRevision model.Revision,
	cardKind string,
	validate castValidator,
	entryOrientation model.CardOrientation,
) error {
	return castCardWithPlan(
		state,
		catalog,
		actingPlayerID,
		cardID,
		model.CastPaymentPlan{Payment: payment},
		expectedRevision,
		cardKind,
		validate,
		entryOrientation,
	)
}

type castAetherProduction struct {
	cardID  model.MatchCardID
	element model.Element
	amount  int
	token   bool
}

func castCardWithPlan(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	plan model.CastPaymentPlan,
	expectedRevision model.Revision,
	cardKind string,
	validate castValidator,
	entryOrientation model.CardOrientation,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Revision != expectedRevision {
		return fmt.Errorf("expected revision %d, current revision is %d", expectedRevision, state.Revision)
	}
	playerIndex := -1
	for index, player := range state.Players {
		if player.ID == actingPlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("active player not found")
	}
	productions, prospectivePool, err := validateCastAetherSources(
		state,
		catalog,
		actingPlayerID,
		playerIndex,
		plan.SourceCardIDs,
	)
	if err != nil {
		return fmt.Errorf("error preparing Aether sources: %w", err)
	}
	prospective := *state
	prospective.Players[playerIndex].Aether = prospectivePool
	err = validate(&prospective, catalog, actingPlayerID, cardID, plan.Payment)
	if err != nil {
		return fmt.Errorf("error casting %s: %w", cardKind, err)
	}
	nonActingPlayerID := state.Players[(1 - playerIndex)].ID
	if nonActingPlayerID == "" || nonActingPlayerID == actingPlayerID {
		return fmt.Errorf("opposing player was not found")
	}
	cardIndex := -1
	for index, ID := range state.Players[playerIndex].Hand {
		if ID == cardID {
			cardIndex = index
			break
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card %q not found in %q player's hand", cardID, actingPlayerID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q was not found in card instances", cardID)
	}
	applyCastAetherProductions(state, playerIndex, productions)
	payAether(&state.Players[playerIndex].Aether, plan.Payment)
	hand := state.Players[playerIndex].Hand
	hand = slices.Delete(hand, cardIndex, cardIndex+1)
	state.Players[playerIndex].Hand = hand
	instance.Face = model.CardFaceUp
	instance.Controller = actingPlayerID
	state.CardInstances[cardID] = instance
	link := model.ChaseLink{
		ID:               state.NextLinkID,
		Controller:       actingPlayerID,
		SourceCardID:     cardID,
		Kind:             model.ChaseLinkCardPlay,
		EntryOrientation: entryOrientation,
	}
	state.ChaseLinks = append(state.ChaseLinks, link)
	state.NextLinkID++
	state.PassCount = 0
	state.PriorityHolder = actingPlayerID
	state.Revision++
	return nil
}

func validateCastAetherSources(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	playerIndex int,
	sourceIDs []model.MatchCardID,
) ([]castAetherProduction, model.AetherPool, error) {
	pool := state.Players[playerIndex].Aether
	productions := make([]castAetherProduction, 0, len(sourceIDs))
	seen := make(map[model.MatchCardID]struct{}, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		if _, duplicate := seen[sourceID]; duplicate {
			return nil, model.AetherPool{}, fmt.Errorf("Aether source %q was selected more than once", sourceID)
		}
		seen[sourceID] = struct{}{}
		instance, exists := state.CardInstances[sourceID]
		if !exists {
			return nil, model.AetherPool{}, fmt.Errorf("Aether source %q was not found", sourceID)
		}
		production := castAetherProduction{cardID: sourceID}
		switch {
		case instance.CardID == model.CasterTokenCardID:
			if err := rules.ValidateUseCasterToken(state, actingPlayerID, sourceID); err != nil {
				return nil, model.AetherPool{}, fmt.Errorf("Caster Token %q: %w", sourceID, err)
			}
			production.amount = 1
			production.token = true
			pool.NonElemental++
		case instance.Face == model.CardFaceDown:
			if err := rules.ValidateGenerateNonElementalAether(state, actingPlayerID, sourceID); err != nil {
				return nil, model.AetherPool{}, fmt.Errorf("face-down Caster %q: %w", sourceID, err)
			}
			production.amount = 1
			pool.NonElemental++
		default:
			element, amount, err := rules.ValidateGenerateCasterAether(state, catalog, actingPlayerID, sourceID)
			if err != nil {
				return nil, model.AetherPool{}, fmt.Errorf("face-up Caster %q: %w", sourceID, err)
			}
			production.element = element
			production.amount = amount
			if err := addAether(&pool, element, amount); err != nil {
				return nil, model.AetherPool{}, err
			}
		}
		productions = append(productions, production)
	}
	return productions, pool, nil
}

func applyCastAetherProductions(state *model.MatchState, playerIndex int, productions []castAetherProduction) {
	player := &state.Players[playerIndex]
	for _, production := range productions {
		if production.token {
			delete(state.CardInstances, production.cardID)
			cardIndex := slices.Index(player.CasterZone, production.cardID)
			player.CasterZone = slices.Delete(player.CasterZone, cardIndex, cardIndex+1)
			player.Aether.NonElemental++
			continue
		}
		instance := state.CardInstances[production.cardID]
		instance.Orientation = model.OrientationRested
		state.CardInstances[production.cardID] = instance
		if production.element == "" {
			player.Aether.NonElemental += production.amount
			continue
		}
		_ = addAether(&player.Aether, production.element, production.amount)
	}
}

func addAether(pool *model.AetherPool, element model.Element, amount int) error {
	var destination *int
	switch element {
	case model.ElementAes:
		destination = &pool.Aes
	case model.ElementAqua:
		destination = &pool.Aqua
	case model.ElementIgnus:
		destination = &pool.Ignus
	case model.ElementLuna:
		destination = &pool.Luna
	case model.ElementSilva:
		destination = &pool.Silva
	case model.ElementSolis:
		destination = &pool.Solis
	case model.ElementTerra:
		destination = &pool.Terra
	case model.ElementVoid:
		destination = &pool.Void
	default:
		return fmt.Errorf("unsupported Aether element %q", element)
	}
	*destination += amount
	return nil
}
