package distribution

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestClientFetchesAndVerifiesCatalog(t *testing.T) {
	database, err := EncodeCards([]cards.Card{{ID: "1", Name: "One"}})
	if err != nil {
		t.Fatal(err)
	}
	pointer := CatalogPointer{
		SchemaVersion: SchemaVersion, CatalogVersion: "v1",
		ReleaseURL: "https://assets.test/catalog/v1/release.json",
	}
	release := validReleaseManifest()
	release.Database.URL = "https://assets.test/catalog/v1/cards.json"
	release.Database.Size = int64(len(database))
	release.Database.SHA256 = SHA256(database)
	release.TabletopSimulator.ManifestURL = "https://assets.test/catalog/v1/tts/manifest.json"
	tts := validTTSManifest()

	responses := map[string][]byte{
		"https://assets.test/catalog/current.json":         marshalTestJSON(t, pointer),
		"https://assets.test/catalog/v1/release.json":      marshalTestJSON(t, release),
		"https://assets.test/catalog/v1/cards.json":        database,
		"https://assets.test/catalog/v1/tts/manifest.json": marshalTestJSON(t, tts),
	}
	client := Client{
		PointerURL: "https://assets.test/catalog/current.json",
		UserAgent:  "test-agent",
		HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Header.Get("User-Agent") != "test-agent" {
				t.Errorf("User-Agent = %q", request.Header.Get("User-Agent"))
			}
			body, found := responses[request.URL.String()]
			if !found {
				return testResponse(request, http.StatusNotFound, []byte("missing")), nil
			}
			return testResponse(request, http.StatusOK, body), nil
		})},
	}

	gotPointer, gotRelease, err := client.FetchCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPointer.CatalogVersion != "v1" || gotRelease.CatalogVersion != "v1" {
		t.Fatalf("unexpected release: %#v %#v", gotPointer, gotRelease)
	}
	repository, downloaded, err := client.FetchDatabase(context.Background(), gotRelease)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := repository.FindByID("1"); !found || string(downloaded) != string(database) {
		t.Fatal("verified database was not returned")
	}
	if _, err := client.FetchTTSManifest(context.Background(), gotRelease); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsDatabaseIntegrityFailure(t *testing.T) {
	release := validReleaseManifest()
	release.Database.URL = "https://assets.test/cards.json"
	release.Database.Size = 7
	release.Database.SHA256 = strings.Repeat("a", 64)
	client := Client{HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return testResponse(request, http.StatusOK, []byte("changed")), nil
	})}}
	if _, _, err := client.FetchDatabase(context.Background(), release); err == nil {
		t.Fatal("FetchDatabase() accepted a bad digest")
	}
}

func TestClientRejectsCatalogVersionMismatch(t *testing.T) {
	pointer := CatalogPointer{
		SchemaVersion: SchemaVersion, CatalogVersion: "v2",
		ReleaseURL: "https://assets.test/release.json",
	}
	release := validReleaseManifest()
	responses := [][]byte{marshalTestJSON(t, pointer), marshalTestJSON(t, release)}
	index := 0
	client := Client{
		PointerURL: "https://assets.test/current.json",
		HTTP: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			response := testResponse(request, http.StatusOK, responses[index])
			index++
			return response, nil
		})},
	}
	if _, _, err := client.FetchCurrent(context.Background()); err == nil {
		t.Fatal("FetchCurrent() accepted mismatched versions")
	}
}

func TestCardImageURL(t *testing.T) {
	release := validReleaseManifest()
	got, err := CardImageURL(release, " card/1 ")
	if err != nil {
		t.Fatal(err)
	}
	if got != release.Images.BaseURL+"card%2F1.png" {
		t.Fatalf("CardImageURL() = %q", got)
	}
}

func marshalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testResponse(request *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Request:    request,
	}
}
