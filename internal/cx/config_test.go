package cx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigCreatesDefaultsWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned %v", err)
	}

	if cfg.DefaultProvider != ProviderAnthropic {
		t.Errorf("DefaultProvider = %q, a fresh install must not route through a gateway", cfg.DefaultProvider)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file was not written: %v", err)
	}

	reread, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("re-reading the written config returned %v", err)
	}
	if reread.Providers[ProviderAkashML].BaseURL != cfg.Providers[ProviderAkashML].BaseURL {
		t.Error("the config cx writes does not round-trip through its own parser")
	}
}

func TestLoadConfigFillsOmittedFieldsFromDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	write(t, path, `{
	  "defaultProvider": "akashml",
	  "providers": { "akashml": { "models": { "haiku": "a-smaller-model" } } }
	}`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned %v", err)
	}

	akash := cfg.Providers[ProviderAkashML]
	if cfg.DefaultProvider != ProviderAkashML {
		t.Errorf("DefaultProvider = %q, want the file's value", cfg.DefaultProvider)
	}
	if akash.Models.Haiku != "a-smaller-model" {
		t.Errorf("Models.Haiku = %q, want the file's value", akash.Models.Haiku)
	}
	if akash.Models.Sonnet != DefaultConfig().Providers[ProviderAkashML].Models.Sonnet {
		t.Errorf("Models.Sonnet = %q, an omitted tier must fall back to the default", akash.Models.Sonnet)
	}
	if akash.BaseURL == "" || akash.KeychainService == "" {
		t.Error("omitting baseUrl or keychainService left the provider unusable")
	}
}

func TestLoadConfigAcceptsAnExtraProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	write(t, path, `{"providers": {"other": {"baseUrl": "https://example.test", "keychainService": "other"}}}`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned %v", err)
	}

	if _, err := cfg.Resolve("other"); err != nil {
		t.Errorf("Resolve(other) returned %v, a hand-added gateway should just work", err)
	}
	if _, ok := cfg.Providers[ProviderAkashML]; !ok {
		t.Error("adding a provider dropped the built-in akashml entry")
	}
}

func TestLoadConfigReportsUnparsableFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	write(t, path, "{not json")

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig silently ignored a broken config file")
	}
}

func write(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
