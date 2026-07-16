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
