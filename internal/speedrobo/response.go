package speedrobo

type SearchResponse struct {
	Success bool       `json:"success"`
	Data    SearchData `json:"data"`
}

type SearchData struct {
	Cards   []CardResponse `json:"cards"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	Pages   int            `json:"pages"`
	PerPage int            `json:"per_page"`
}

type CardResponse struct {
	ID          string `json:"id"`
	CardKey     string `json:"card_key"`
	ImageURL    string `json:"image_url"`
	Expansion   string `json:"expansion_name"`
	PlayTesting string `json:"is_playtesting"`
	Favorite    bool   `json:"is_favorite"`
}

type CardDetailResponse struct {
	Success bool           `json:"success"`
	Data    CardDetailData `json:"data"`
}

type CardDetailData struct {
	Card CardDetail `json:"card"`
}

type CardDetail struct {
	ID            string      `json:"id"`
	ExpansionID   string      `json:"expansion_id"`
	CardKey       string      `json:"card_key"`
	ImageURL      string      `json:"image_url"`
	CreatedAt     string      `json:"created_at"`
	ExpansionName string      `json:"expansion_name"`
	game_id       string      `json:"game_id"`
	IsPlaytesting string      `json:"is_playtesting"`
	Fields        []CardField `json:"fields"`
	IsFavorite    bool        `json:"is_favorite"`
}

type CardField struct {
	Label string `json:"field_label"`
	Value string `json:"value"`
}
