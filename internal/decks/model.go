package decks

const (
	MaxMainCards = 50
	MaxSideCards = 12
	MaxCopies    = 4
)

type DeckEntry struct {
	CardID   string `json:"card_id"`
	Quantity int    `json:"quantity"`
}

type Deck struct {
	SchemaVersion int         `json:"schema_version"`
	Name          string      `json:"name"`
	MainDeck      []DeckEntry `json:"main_deck"`
	SideDeck      []DeckEntry `json:"side_deck"`
	MainOrder	  []string    `json:"main_order,omitempty"`
	SideOrder	  []string    `json:"side_order,omitempty"`
}
