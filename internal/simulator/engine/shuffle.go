package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func shuffleDeck(deck []model.MatchCardID, random RandomSource) error {
	for index := len(deck) - 1; index > 0; index-- {
		swapIndex := random.RandInt(index + 1)
		if swapIndex < 0 || swapIndex > index {
			return fmt.Errorf("%d is an invalid index value. Must be between 0 and %d", swapIndex, index)
		}
		deck[index], deck[swapIndex] = deck[swapIndex], deck[index]
	}
	return nil
}
