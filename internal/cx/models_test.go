package cx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchModelsReadsTheProvidersOwnList(t *testing.T) {
	var gotPath, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[{"id":"a--one","display_name":"One"},{"id":"b--two"}]}`))
	}))
	defer server.Close()

	models, err := FetchModels(Provider{BaseURL: server.URL}, "akml-EXAMPLE-token")
	if err != nil {
		t.Fatalf("FetchModels returned %v", err)
	}

	if gotPath != "/v1/models" {
		t.Errorf("asked for %q, want /v1/models", gotPath)
	}
	if gotAuth != "Bearer akml-EXAMPLE-token" {
		t.Errorf("Authorization = %q, a gateway that gates its model list would refuse", gotAuth)
	}
	if len(models) != 2 || models[0].ID != "a--one" {
		t.Fatalf("got %v, want both models in order", models)
	}
	if models[0].Name() != "One" || models[1].Name() != "b--two" {
		t.Errorf("Name() = %q, %q; an entry without a display name should fall back to its id", models[0].Name(), models[1].Name())
	}
}

func TestFetchModelsReportsAnUnusableAnswer(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"an error status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }},
		{"an empty list", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"data":[]}`)) }},
		{"not json", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`<html>`)) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			if _, err := FetchModels(Provider{BaseURL: server.URL}, ""); err == nil {
				t.Error("FetchModels reported success, so cx would offer an empty or bogus list")
			}
		})
	}
}

func TestFetchModelsJoinsTheBaseURLWithoutDoublingTheSlash(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"data":[{"id":"x"}]}`))
	}))
	defer server.Close()

	if _, err := FetchModels(Provider{BaseURL: server.URL + "/"}, ""); err != nil {
		t.Fatalf("FetchModels returned %v", err)
	}
	if gotPath != "/v1/models" {
		t.Errorf("asked for %q, want /v1/models even when baseUrl ends in a slash", gotPath)
	}
}

func TestTierNamedAcceptsTheSpellingsPeopleType(t *testing.T) {
	for _, spelling := range []string{"smallFast", "smallfast", "small-fast", "small_fast", "small", "fast", "FAST"} {
		if tier, ok := TierNamed(spelling); !ok || tier != "smallFast" {
			t.Errorf("TierNamed(%q) = %q, %v; want smallFast", spelling, tier, ok)
		}
	}
	if _, ok := TierNamed("turbo"); ok {
		t.Error("TierNamed accepted a tier that does not exist, which would silently write nothing")
	}
}

func TestModelsSetAndTiersUsing(t *testing.T) {
	m := Models{}
	m.Set("opus", "big")
	m.Set("sonnet", "big")
	m.Set("haiku", "small")

	if got := m.TiersUsing("big"); len(got) != 2 || got[0] != "opus" || got[1] != "sonnet" {
		t.Errorf("TiersUsing(big) = %v, want opus and sonnet in tier order", got)
	}
	if got := m.TiersUsing("nothing"); len(got) != 0 {
		t.Errorf("TiersUsing(nothing) = %v, want none", got)
	}
}

func TestConversationWindowTakesTheSmallestOfTheTalkingTiers(t *testing.T) {
	models := []Model{
		{ID: "huge", ContextLength: 1 << 20},
		{ID: "small", ContextLength: 131072},
		{ID: "unknown"},
	}

	tests := []struct {
		name       string
		opus       string
		sonnet     string
		haiku      string
		wantWindow int
	}{
		{name: "both tiers on the same model", opus: "huge", sonnet: "huge", wantWindow: 1 << 20},
		{name: "mixed tiers clamp to the smaller", opus: "huge", sonnet: "small", wantWindow: 131072},
		{name: "a short cheap tier does not drag the window down", opus: "huge", sonnet: "huge", haiku: "small", wantWindow: 1 << 20},
		{name: "an unknown window is skipped rather than counted as zero", opus: "unknown", sonnet: "huge", wantWindow: 1 << 20},
		{name: "nothing known means say nothing", opus: "unknown", sonnet: "unknown", wantWindow: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configured := Models{Opus: tt.opus, Sonnet: tt.sonnet, Haiku: tt.haiku}

			if got := ConversationWindow(configured, models); got != tt.wantWindow {
				t.Errorf("got %d, want %d", got, tt.wantWindow)
			}
		})
	}
}

func TestEnrichmentMatchesAcrossTheTwoIdSpellings(t *testing.T) {
	catalogue := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"zai-org/GLM-5.3","context_length":1048576,"input_modalities":["text"]},
		                          {"id":"Qwen/Qwen3.8-27B","context_length":262144,"input_modalities":["text","image"]}]}`))
	}))
	defer catalogue.Close()

	list := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"zai-org--GLM-5.3"},{"id":"Qwen--Qwen3.8-27B"},{"id":"not--in-catalogue"}]}`))
	}))
	defer list.Close()

	models, err := FetchModels(Provider{BaseURL: list.URL, CatalogueURL: catalogue.URL}, "akml-EXAMPLE-token")
	if err != nil {
		t.Fatalf("FetchModels returned %v", err)
	}

	if models[0].ContextLength != 1048576 {
		t.Errorf("GLM context = %d; the listings spell ids differently and must still match", models[0].ContextLength)
	}
	if models[0].TakesImages() {
		t.Error("GLM reported as taking images")
	}
	if !models[1].TakesImages() {
		t.Error("Qwen reported as not taking images")
	}
	if models[2].ContextLength != 0 {
		t.Error("a model absent from the catalogue was given a window it never declared")
	}
}

func TestEnrichmentFailureLeavesTheListingUsable(t *testing.T) {
	list := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"a--model"}]}`))
	}))
	defer list.Close()

	models, err := FetchModels(Provider{BaseURL: list.URL, CatalogueURL: "http://127.0.0.1:1/nothing"}, "")

	if err != nil {
		t.Fatalf("an unreachable catalogue broke the listing: %v", err)
	}
	if len(models) != 1 || models[0].ID != "a--model" {
		t.Errorf("got %v, want the plain listing to survive", models)
	}
}
