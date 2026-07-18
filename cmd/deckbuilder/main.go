// Package main launches the Caster's Compendium deckbuilder.
package main

import deckbuilder "github.com/HybridUofA/casters-compendium/internal/deckbuilder/app"

// main delegates desktop startup to the deckbuilder application package.
func main() {
	deckbuilder.Run()
}
