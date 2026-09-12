package cx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	ProviderAnthropic = "anthropic"
	ProviderAkashML   = "akashml"
)

type Models struct {
	Opus      string `json:"opus"`
	Sonnet    string `json:"sonnet"`
	Haiku     string `json:"haiku"`
	SmallFast string `json:"smallFast"`
}

type Provider struct {
	BaseURL         string `json:"baseUrl"`
	KeychainService string `json:"keychainService"`
	// TokenPrefix, when set, is what this provider's keys start with. cx checks
	// it so that a mistyped credential fails at the prompt rather than as a 401.
	TokenPrefix string `json:"tokenPrefix"`
	// BehavesAs names the Claude model whose capabilities Claude Code should
	// assume for this provider's models. Without it, Claude Code refuses an id
	// that is absent from its own catalog.
	BehavesAs string `json:"behavesAs"`
	// CatalogueURL is a richer model listing than the Anthropic-shaped one,
	// carrying context windows and modalities.
	CatalogueURL string `json:"catalogueUrl"`
	// MaxContextTokens is the window Claude Code should work to. Without it the
	// session is held to whatever BehavesAs implies, which for an open model with
	// a long window is far short of what it can take.
	MaxContextTokens int `json:"maxContextTokens"`
	// APITimeoutMS raises the per-request deadline, since open models served
	// behind a gateway are often much slower to first token than Claude is.
	APITimeoutMS int    `json:"apiTimeoutMs"`
	Models       Models `json:"models"`
}

type Config struct {
	DefaultProvider string              `json:"defaultProvider"`
	Providers       map[string]Provider `json:"providers"`
}

func DefaultConfig() Config {
	return Config{
		DefaultProvider: ProviderAnthropic,
		Providers: map[string]Provider{
			ProviderAkashML: {
				BaseURL:         "https://api.akashml.com/anthropic",
				KeychainService: ProviderAkashML,
				TokenPrefix:     "akml-",
				BehavesAs:       "claude-sonnet-5",
				CatalogueURL:    "https://api.akashml.com/v1/models",
				APITimeoutMS:    3000000,
				Models: Models{
					Opus:      "zai-org--GLM-5.3",
					Sonnet:    "zai-org--GLM-5.3",
					Haiku:     "openai--gpt-oss-20b",
					SmallFast: "openai--gpt-oss-20b",
				},
			},
		},
	}
}

// LoadConfig reads path, creating it with defaults when absent. Fields the file
// omits fall back to their default, so a hand-trimmed config never yields an
// empty base URL or model id.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, writeConfig(path, cfg)
	}
	if err != nil {
		return cfg, err
	}

	var onDisk Config
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", path, err)
	}

	return merge(DefaultConfig(), onDisk), nil
}

func merge(base, over Config) Config {
	if over.DefaultProvider != "" {
		base.DefaultProvider = over.DefaultProvider
	}
	for name, p := range over.Providers {
		base.Providers[name] = mergeProvider(base.Providers[name], p)
	}
	return base
}

func mergeProvider(base, over Provider) Provider {
	if over.BaseURL != "" {
		base.BaseURL = over.BaseURL
	}
	if over.KeychainService != "" {
		base.KeychainService = over.KeychainService
	}
	if over.TokenPrefix != "" {
		base.TokenPrefix = over.TokenPrefix
	}
	if over.BehavesAs != "" {
		base.BehavesAs = over.BehavesAs
	}
	if over.CatalogueURL != "" {
		base.CatalogueURL = over.CatalogueURL
	}
	if over.MaxContextTokens != 0 {
		base.MaxContextTokens = over.MaxContextTokens
	}
	if over.APITimeoutMS != 0 {
		base.APITimeoutMS = over.APITimeoutMS
	}
	if over.Models.Opus != "" {
		base.Models.Opus = over.Models.Opus
	}
	if over.Models.Sonnet != "" {
		base.Models.Sonnet = over.Models.Sonnet
	}
	if over.Models.Haiku != "" {
		base.Models.Haiku = over.Models.Haiku
	}
	if over.Models.SmallFast != "" {
		base.Models.SmallFast = over.Models.SmallFast
	}
	return base
}

func writeConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// ProviderNames returns the configured gateways in a stable order.
func (c Config) ProviderNames() []string {
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Save writes the configuration back, so that cx can edit it in place instead
// of asking a person to hand-edit JSON.
func Save(path string, cfg Config) error {
	return writeConfig(path, cfg)
}
