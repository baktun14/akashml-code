package cx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`

	// Filled in from the provider's catalogue when it publishes one. The
	// Anthropic-shaped model list carries none of this.
	ContextLength   int
	InputModalities []string
}

func (m Model) TakesImages() bool {
	for _, kind := range m.InputModalities {
		if kind == "image" {
			return true
		}
	}
	return false
}

func (m Model) Name() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.ID
}

// FetchModels asks the provider which models it serves, so the ids offered are
// always the ones it will actually accept rather than whatever its docs say.
func FetchModels(p Provider, token string) ([]Model, error) {
	url := strings.TrimSuffix(p.BaseURL, "/") + "/v1/models"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("anthropic-version", "2023-06-01")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("asking %s for its models: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", url, resp.Status)
	}

	var body struct {
		Data []Model `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("reading the model list from %s: %w", url, err)
	}
	if len(body.Data) == 0 {
		return nil, fmt.Errorf("%s returned no models", url)
	}

	enrichFromCatalogue(body.Data, p, token)
	return body.Data, nil
}

// catalogueEntry is the OpenAI-shaped model record, which carries the context
// window and modalities that the Anthropic-shaped list leaves out.
type catalogueEntry struct {
	ID              string   `json:"id"`
	ContextLength   int      `json:"context_length"`
	InputModalities []string `json:"input_modalities"`
}

// enrichFromCatalogue fills in what the provider's richer listing knows. It is
// best effort: a provider that publishes no catalogue simply leaves the extra
// fields empty rather than failing the listing.
func enrichFromCatalogue(models []Model, p Provider, token string) {
	if p.CatalogueURL == "" {
		return
	}

	req, err := http.NewRequest(http.MethodGet, p.CatalogueURL, nil)
	if err != nil {
		return
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var body struct {
		Data []catalogueEntry `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil {
		return
	}

	// The two listings spell the same model differently, so match on a form that
	// ignores the separator rather than on the raw id.
	byKey := make(map[string]catalogueEntry, len(body.Data))
	for _, entry := range body.Data {
		byKey[modelKey(entry.ID)] = entry
	}
	for i := range models {
		if entry, ok := byKey[modelKey(models[i].ID)]; ok {
			models[i].ContextLength = entry.ContextLength
			models[i].InputModalities = entry.InputModalities
		}
	}
}

func modelKey(id string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(id, "--", "/"), "_", "-"))
}

// Tiers are the model slots Claude Code resolves, in the order cx displays them.
var Tiers = []string{"opus", "sonnet", "haiku", "smallFast"}

// TierNamed resolves the spellings a person is likely to type for a tier.
func TierNamed(name string) (string, bool) {
	switch strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", ""), "_", "")) {
	case "opus":
		return "opus", true
	case "sonnet":
		return "sonnet", true
	case "haiku":
		return "haiku", true
	case "smallfast", "small", "fast":
		return "smallFast", true
	}
	return "", false
}

func (m *Models) Get(tier string) string {
	switch tier {
	case "opus":
		return m.Opus
	case "sonnet":
		return m.Sonnet
	case "haiku":
		return m.Haiku
	case "smallFast":
		return m.SmallFast
	}
	return ""
}

func (m *Models) Set(tier, model string) {
	switch tier {
	case "opus":
		m.Opus = model
	case "sonnet":
		m.Sonnet = model
	case "haiku":
		m.Haiku = model
	case "smallFast":
		m.SmallFast = model
	}
}

// TiersUsing lists the tiers pointed at a model, so a listing can show what is
// already in use.
func (m Models) TiersUsing(model string) []string {
	var used []string
	for _, tier := range Tiers {
		if m.Get(tier) == model {
			used = append(used, tier)
		}
	}
	return used
}

// ConversationWindow is the smallest window among the models that carry the
// conversation. Claude Code works to one number for a session, so it has to be
// one every model that might answer can actually accept.
func ConversationWindow(configured Models, models []Model) int {
	lengths := make(map[string]int, len(models))
	for _, m := range models {
		lengths[m.ID] = m.ContextLength
	}

	smallest := 0
	for _, tier := range []string{"opus", "sonnet"} {
		length := lengths[configured.Get(tier)]
		if length == 0 {
			continue
		}
		if smallest == 0 || length < smallest {
			smallest = length
		}
	}
	return smallest
}
