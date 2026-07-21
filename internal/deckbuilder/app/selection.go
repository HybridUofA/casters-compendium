package deckbuilder

import (
	"slices"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type SelectedState struct {
	zone    decks.Zone
	indices map[int]struct{}
}

func (selection *SelectedState) Clear() {
	selection.zone = ""
	selection.indices = make(map[int]struct{})
}

func (selection *SelectedState) Contains(zone decks.Zone, index int) bool {
	if selection.zone != zone {
		return false
	}
	_, found := selection.indices[index]
	return found
}

func (selection *SelectedState) SortedIndices() []int {
	sorted := make([]int, 0, selection.Count())
	for index := range selection.indices {
		sorted = append(sorted, index)
	}
	slices.Sort(sorted)
	return sorted
}

func (selection *SelectedState) Count() int {
	return len(selection.indices)
}

func (selection *SelectedState) Toggle(zone decks.Zone, index int) {
	if index < 0 {
		return
	}
	if selection == nil {
		return
	}
	if selection.Count() == 0 || selection.zone != zone {
		selection.Clear()
		selection.zone = zone
	}

	if selection.Contains(zone, index) {
		delete(selection.indices, index)
		if selection.Count() == 0 {
			selection.Clear()
		}
		return
	}
	selection.indices[index] = struct{}{}
}
