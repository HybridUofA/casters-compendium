package model

import "strings"

// MarkCardKnown records that viewerID has seen cardID's identity.
func MarkCardKnown(state *MatchState, viewerID PlayerID, cardID MatchCardID) {
	if state == nil || strings.TrimSpace(string(viewerID)) == "" || strings.TrimSpace(string(cardID)) == "" {
		return
	}
	if state.KnownCards == nil {
		state.KnownCards = make(map[PlayerID]map[MatchCardID]struct{})
	}
	known := state.KnownCards[viewerID]
	if known == nil {
		known = make(map[MatchCardID]struct{})
		state.KnownCards[viewerID] = known
	}
	known[cardID] = struct{}{}
}

// ViewerKnowsCard reports whether viewerID has persistent knowledge of cardID.
func ViewerKnowsCard(state *MatchState, viewerID PlayerID, cardID MatchCardID) bool {
	if state == nil {
		return false
	}
	known, ok := state.KnownCards[viewerID]
	if !ok {
		return false
	}
	_, found := known[cardID]
	return found
}
