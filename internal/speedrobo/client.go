package speedrobo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"
)

const (
	ajaxURL     = "https://speedrobogames.com/wp-admin/admin-ajax.php"
	databaseURL = "https://speedrobogames.com/card-database"
)

// NewClient constructs the shared timeout and cookie behavior required by Speedrobo endpoints.
func NewClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
	}

	return client, nil
}

// ExtractPageConfig parses the AJAX endpoint and nonce embedded in the card-database page.
func ExtractPageConfig(pageHTML []byte) (PageConfig, error) {
	var config PageConfig

	markers := [][]byte{
		[]byte("window.SRC ="),
		[]byte("var SRC ="),
	}

	var remainingHTML []byte

	for _, marker := range markers {
		markerPosition := bytes.Index(pageHTML, marker)
		if markerPosition == -1 {
			continue
		}

		remainingHTML = pageHTML[markerPosition+len(marker):]
		break
	}

	if remainingHTML == nil {
		return config, fmt.Errorf("could not find SRC configuration")
	}

	jsonStart := bytes.IndexByte(remainingHTML, '{')
	if jsonStart == -1 {
		return config, fmt.Errorf("SRC configuration has no opening brace")
	}

	remainingHTML = remainingHTML[jsonStart:]

	jsonEnd := bytes.Index(remainingHTML, []byte("};"))
	if jsonEnd == -1 {
		return config, fmt.Errorf("SRC configuration has no closing brace")
	}

	configJSON := remainingHTML[:jsonEnd+1]

	if err := json.Unmarshal(configJSON, &config); err != nil {
		return config, fmt.Errorf("decode SRC configuration: %w", err)
	}

	if strings.TrimSpace(config.Nonce) == "" {
		return config, fmt.Errorf("SRC configuration contains an empty nonce")
	}

	if strings.TrimSpace(config.AjaxURL) == "" {
		return config, fmt.Errorf("SRC configuration contains an empty AJAX URL")
	}

	return config, nil
}

// FetchDatabasePage retrieves the public HTML page containing runtime API configuration.
func FetchDatabasePage(client *http.Client) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("HTTP client cannot be nil")
	}

	req, err := http.NewRequest(
		http.MethodGet,
		databaseURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create database-page request: %w", err)
	}

	req.Header.Set(
		"User-Agent",
		"casters-compendium/0.1",
	)
	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml",
	)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch database page: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"database page returned %s",
			res.Status,
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"read database page: %w",
			err,
		)
	}

	return body, nil
}

// FetchPageConfig downloads the database page and extracts its API configuration.
func FetchPageConfig(client *http.Client) (PageConfig, error) {
	var config PageConfig

	pageHTML, err := FetchDatabasePage(client)
	if err != nil {
		return config, fmt.Errorf("fetch page configuration: %w", err)
	}

	config, err = ExtractPageConfig(pageHTML)
	if err != nil {
		return config, fmt.Errorf("extract page configuration: %w", err)
	}

	return config, nil
}

// FetchPage retrieves one paginated set of card-summary records.
func FetchPage(
	client *http.Client,
	ajaxURL string,
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
	req.Header.Set("User-Agent", "casters-compendium/0.1")
	req.Header.Set("Origin", "https://speedrobogames.com")
	req.Header.Set("Referer", databaseURL)

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
		return result, fmt.Errorf("unexpected HTTP status %s: %s", res.Status, shortBody(responseBody))
	}

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return result, fmt.Errorf("decode response JSON: %w; body: %s", err, shortBody(responseBody))
	}

	if !result.Success {
		return result, fmt.Errorf("Speedrobo returned success=false: %s", shortBody(responseBody))
	}

	return result, nil
}

// FetchCardDetails retrieves the complete field set for one card identifier.
func FetchCardDetails(
	client *http.Client,
	ajaxURL string,
	nonce string,
	cardID string,
) (CardDetailResponse, error) {
	var result CardDetailResponse
	if client == nil {
		return result, fmt.Errorf("HTTP client cannot be nil")
	}

	if strings.TrimSpace(ajaxURL) == "" {
		return result, fmt.Errorf("AJAX URL cannot be empty")
	}

	if strings.TrimSpace(nonce) == "" {
		return result, fmt.Errorf("nonce cannot be empty")
	}

	if strings.TrimSpace(cardID) == "" {
		return result, fmt.Errorf("card ID cannot be empty")
	}

	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

	fields := []struct {
		name  string
		value string
	}{
		{name: "nonce", value: nonce},
		{name: "action", value: "src_get_card"},
		{name: "card_id", value: cardID},
	}

	for _, field := range fields {
		if err := writer.WriteField(field.name, field.value); err != nil {
			return result, fmt.Errorf(
				"write multipart field %q, %w",
				field.name,
				err,
			)
		}
	}

	if err := writer.Close(); err != nil {
		return result, fmt.Errorf("finish multipart body: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		ajaxURL,
		&requestBody,
	)
	if err != nil {
		return result, fmt.Errorf("create card-details request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "casters-compendium/0.1")
	req.Header.Set("Origin", "https://speedrobogames.com")
	req.Header.Set("Referer", databaseURL)

	res, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("read card-details response: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response body")
	}

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return result, fmt.Errorf(
			"decode card-details response: %w; body: %s",
			err,
			shortBody(responseBody),
		)
	}

	if !result.Success {
		return result, fmt.Errorf(
			"Speedrobo returned success=false: %s",
			shortBody(responseBody),
		)
	}

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf(
			"unexpected HTTP status %s: %s",
			res.Status,
			shortBody(responseBody),
		)
	}

	return result, nil
}

// shortBody trims diagnostic response text to a safe, readable maximum length.
func shortBody(body []byte) string {
	const maximumLength = 300

	text := strings.TrimSpace(string(body))

	if len(text) <= maximumLength {
		return text
	}

	return text[:maximumLength] + "..."
}

// FetchAllCards retrieves every summary page and validates the reported total.
func FetchAllCards(
	client *http.Client,
	config PageConfig,
) ([]CardResponse, error) {
	firstResponse, err := FetchPage(client, config.AjaxURL, config.Nonce, 1)
	if err != nil {
		return nil, fmt.Errorf("error occurred on page 1: %w", err)
	}

	allCards := append(
		[]CardResponse{},
		firstResponse.Data.Cards...,
	)

	for page := 2; page <= firstResponse.Data.Pages; page++ {
		pageResponse, err := FetchPage(
			client,
			config.AjaxURL,
			config.Nonce,
			page,
		)
		if err != nil {
			return nil, fmt.Errorf("error occurred on page %d: %w", page, err)
		}

		if pageResponse.Data.Page != page {
			return nil, fmt.Errorf(
				"requested page %d but received page %d",
				page,
				pageResponse.Data.Page,
			)
		}

		allCards = append(allCards, pageResponse.Data.Cards...)
	}

	if len(allCards) != firstResponse.Data.Total {
		return nil, fmt.Errorf(
			"card count mismatch: expected %d, received %d",
			firstResponse.Data.Total,
			len(allCards),
		)
	}

	return allCards, nil
}

// FetchAllCardDetails sequentially downloads full records for a summary list.
func FetchAllCardDetails(
	client *http.Client,
	config PageConfig,
	summaries []CardResponse,
) ([]CardDetail, error) {
	details := make(
		[]CardDetail,
		0,
		len(summaries),
	)

	for index, summary := range summaries {
		response, err := FetchCardDetails(
			client,
			config.AjaxURL,
			config.Nonce,
			summary.ID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"fetch details for card %q (%s): %w",
				summary.CardKey,
				summary.ID,
				err,
			)
		}
		details = append(details, response.Data.Card)

		fmt.Printf(
			"Fetched card %d/%d: %s\n",
			index+1,
			len(summaries),
			summary.CardKey,
		)

		if index < len(summaries)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return details, nil
}
