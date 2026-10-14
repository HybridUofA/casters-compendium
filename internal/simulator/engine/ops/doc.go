// Package ops provides leaf match-state mutations for printed card effects.
//
// Ops never bump Revision and never enforce turn/priority timing. Engine
// commands and effect recipes own those checks, then call into this package.
//
// Current surface (CR 1.2 §10 Game Actions + catalog verbs):
//
//	Draw, Mill, Discard, Banish, BanishStock
//	ReturnToHand, MoveToHand, ReturnToDeck, Exile, DestroyServant
//	SetOrientation, Switch, SetFace
//	ModifyAttack, ModifyDefense, SetCombatModifiers, SwapCombatModifiers
//	GrantDoubleCorrupt
//	AddAether, SpendAether
//	MoveOrbToHand, PutAsOrb, PutAsCaster, PutAsServant
//	PeekDeckTop, PutDeckTopOnBottom, PutDeckCardOnTop, PutDeckCardOnBottom
//	MoveFromDeckToHand, ReorderDeckTops, ShuffleDeck
//	AttachStock, GainControl, ExchangeControl
//
// Intentionally deferred / not leaf-only:
//
//	CreateToken (catalog + token erase rules)
//	Search prompts (PendingChoice; then MoveFromDeckToHand / PutAs*)
//	Reveal / Look-at knowledge (engine orb peek/reveal; recipe-owned visibility)
//	Cancel chase link, Soul Bond, counters (no catalog use yet)
//	Continuous / replacement / keyword timing (Hubris, United Front, Break, …)
package ops
