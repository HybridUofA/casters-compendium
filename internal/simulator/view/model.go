package view

import (
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// MatchView is the viewer-safe projection of one authoritative match state.
// It must contain only information the identified viewer is permitted to know;
// presentation and network code must not receive the unrestricted MatchState.
type MatchView struct {
	ViewerID    model.PlayerID
	Players     [2]PlayerView
	MatchStatus model.Status
	Revision    model.Revision
	Turn        model.TurnState
}

// PlayerView contains the zones and public counts that may be displayed for
// one player. Cards within a zone are individually projected as CardViews so
// hidden opponent information never needs to reach the user interface.
type PlayerView struct {
	ID                   model.PlayerID
	DeckCount            int
	Aether               model.AetherPool
	Hand                 []CardView
	Orbs                 []CardView
	CasterZone           []CardView
	ServantZone          []CardView
	Graveyard            []CardView
	Exile                []CardView
	OpeningHandFinalized bool
}

// CardView describes one card-like object exactly as the current viewer may
// perceive it. When ShowFace is false, the UI must render the standard card
// back. CardID and MatchID must remain empty when revealing either identifier
// could disclose information the viewer is not permitted to know.
//
// ShowFace describes presentation, not necessarily knowledge. For example, a
// viewer may know the identity of their own face-down Caster while the field
// must still display that object using the card back.
type CardView struct {
	MatchID     model.MatchCardID
	CardID      model.CardID
	Face        model.CardFace
	Orientation model.CardOrientation
	ShowFace    bool
}
