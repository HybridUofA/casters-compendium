package rules

import (
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

type CardCatalog interface {
	FindByID(id string) (gamecards.Card, bool)
}
