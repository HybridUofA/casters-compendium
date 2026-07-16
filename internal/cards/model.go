package cards

type Card struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Subname       string            `json:"subname,omitempty"`
	Type          string            `json:"type"`
	Element       string            `json:"element"`
	Traits        string            `json:"traits,omitempty"`
	CostLevel     string            `json:"cost_level,omitempty"`
	Attack        string            `json:"attack,omitempty"`
	Defense       string            `json:"defense,omitempty"`
	Ability       string            `json:"ability,omitempty"`
	Flavor        string            `json:"flavor,omitempty"`
	Artist        string            `json:"artist,omitempty"`
	CardNumber    string            `json:"card_number"`
	Count         string            `json:"count,omitempty"`
	ImageURL      string            `json:"image_url"`
	Expansion     string            `json:"expansion"`
	IsPlaytesting bool              `json:"is_playtesting"`
	ExtraFields   map[string]string `json:"extra_fields,omittempty"`
}
