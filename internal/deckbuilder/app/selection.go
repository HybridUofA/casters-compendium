package deckbuilder

import (
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type SelectedState struct {
	zone    decks.Zone
	indices map[int]struct{}
}

func (selection *SelectedState) Clear(bool, error) {

}

func (selection *SelectedState) Contains(zone decks.Zone, index int) bool {

}

func (selection *SelectedState) SortedIndices() []int {

}

func (selection *SelectedState) Count() int {

}
