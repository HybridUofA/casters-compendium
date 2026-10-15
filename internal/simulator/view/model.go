package view

import (
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// MatchView is the viewer-safe projection of one authoritative match state.
// It must contain only information the identified viewer is permitted to know;
// presentation and network code must not receive the unrestricted MatchState.
type MatchView struct {
	ViewerID       model.PlayerID
	Players        [2]PlayerView
	MatchStatus    model.Status
	Revision       model.Revision
	Turn           model.TurnState
	FirstPlayer    model.PlayerID
	PriorityHolder model.PlayerID
	PassCount      int
	ChaseLinkCount int
	// ChaseLinks is the public stack (bottom→top). Card identities on Chase are
	// always revealed so both players can see what is resolving.
	ChaseLinks           []ChaseLinkView
	PrioritySequenceOpen bool
	Attack               model.AttackState
	PendingDraw          model.PendingDraw
	PendingBreak         model.PendingBreak
	PendingChoice        model.PendingChoice
	Result               model.MatchResult
	// DisplayNames maps seat/spectator IDs to lobby display names when known.
	DisplayNames map[model.PlayerID]string
	// Cosmetics maps seat IDs to player-chosen card-back and caster-token URLs.
	Cosmetics map[model.PlayerID]PlayerCosmetics
	// Spectator is true when ViewerID is not a match player seat.
	Spectator bool
	// Messages is the shared event log + chat (newest last). Side-channel data
	// filled by the session layer; not derived from MatchState alone.
	Messages []MatchMessage
}

// ChaseLinkView is one public Chase stack entry.
type ChaseLinkView struct {
	ID               model.ChaseLinkID
	Controller       model.PlayerID
	Kind             model.ChaseLinkKind
	EntryOrientation model.CardOrientation
	Card             CardView
}

// MatchMessageKind distinguishes system events from player chat.
type MatchMessageKind string

const (
	MessageKindEvent MatchMessageKind = "event"
	MessageKindChat  MatchMessageKind = "chat"
)

// MatchMessage is one log/chat line visible to all seats (and spectators).
type MatchMessage struct {
	Seq      uint64
	Kind     MatchMessageKind
	PlayerID model.PlayerID // empty for system events
	Text     string
}

// PlayerCosmetics carries optional allowlisted custom-asset URLs for one seat.
type PlayerCosmetics struct {
	CardBackURL    string `json:"card_back_url,omitempty"`
	CasterTokenURL string `json:"caster_token_url,omitempty"`
}

// PlayerView contains the zones and public counts that may be displayed for
// one player. Cards within a zone are individually projected as CardViews so
// hidden opponent information never needs to reach the user interface.
type PlayerView struct {
	ID        model.PlayerID
	DeckCount int
	Aether    model.AetherPool
	// Deck is only populated for the viewer's own library so the UI can offer
	// Cockatrice-style browse/search without revealing the opponent's order.
	Deck                 []CardView
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
	// Owner and HasStock are projected only when the viewer already knows the
	// card object's identity (MatchID is non-empty). They exist so the UI can
	// avoid doomed manual-move submissions without receiving MatchState.
	Owner    model.PlayerID
	HasStock bool
	// GrantedDoubleCorrupt is visible whenever MatchID is known (field cards).
	GrantedDoubleCorrupt bool
	// AttackModifier / DefenseModifier are additive offsets to printed ATK/DEF.
	// Projected whenever MatchID is known so the UI can show live combat stats
	// (Lumielle swaps, Rest buffs, etc.) the way MD/EDOPro do.
	AttackModifier  int
	DefenseModifier int
}
