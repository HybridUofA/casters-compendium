package speedrobo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const (
	ajaxURL     = "https://speedrobogames.com/wp-admin/admin-ajax.php"
	databaseURL = "https://speedrobogames.com/card-database"
)

func FetchPage(
	client *http.Client,
	nonce string,
	page int,
) (SearchResponse, error) {
	var result SearchResponse

	if client == nil {
		return result, fmt.Errorf("HTTP client cannot be nil")
	}

	if strings.TrimSpace(nonce) == "" {
		return result, fmt.Errorf("nonce cannot be empty")
	}

	if page < 1 {
		return result, fmt.Errorf("page must be at least 1")
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fields := []struct {
		name  string
		value string
	}{
		{name: "nonce", value: nonce},
		{name: "action", value: "src_search_cards"},
		{name: "game_id", value: "1"},
		{name: "query", value: ""},
		{name: "field_key", value: "__all__"},
		{name: "page", value: strconv.Itoa(page)},
	}

	for _, field := range fields {
		if err := writer.WriteField(field.name, field.value); err != nil {
			return result, fmt.Errorf("write multipart field %q: %w", field.name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return result, fmt.Errorf("finish multipart body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, ajaxURL, &requestBody)
	if err != nil {
		return result, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "caster-chronicles-deckbuilder/0.1")
	req.Header.Set("Origin", "https://speedrobogames.com")
	req.Header.Set("Referrer", databaseURL)

	res, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return result, fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected HTTP status %s", res.Status, shortBody(responseBody))
	}

	return result, nil
}

func shortBody(body []byte) string {
	const maximumLength = 300

	text := strings.TrimSpace(string(body))

	if len(text) <= maximumLength {
		return text
	}

	return text[:maximumLength] + "..."
}
