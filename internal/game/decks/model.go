// Package decks defines deck state and simulator-safe deck-building rules.
package decks

type DeckEntry struct {
	CardID   string `json:"card_id"`
	Quantity int    `json:"quantity"`
}

type Deck struct {
	SchemaVersion  int         `json:"schema_version"`
	Name           string      `json:"name"`
	MainDeck       []DeckEntry `json:"main_deck"`
	SideDeck       []DeckEntry `json:"side_deck"`
	MainOrder      []string    `json:"main_order,omitempty"`
	SideOrder      []string    `json:"side_order,omitempty"`
	TTSCardBackURL string      `json:"tts_card_back_url,omitempty"`
}
