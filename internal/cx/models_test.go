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

	models, err := FetchModels(Provider{BaseURL: server.URL}, "akml-token")
	if err != nil {
		t.Fatalf("FetchModels returned %v", err)
	}

	if gotPath != "/v1/models" {
		t.Errorf("asked for %q, want /v1/models", gotPath)
	}
	if gotAuth != "Bearer akml-token" {
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
