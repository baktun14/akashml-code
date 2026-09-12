package cx

import (
	"strings"
	"testing"
)

func testProvider() Provider {
	return Provider{
		BaseURL:         "https://api.akashml.com/anthropic",
		KeychainService: "akashml",
		Models: Models{
			Opus:      "opus-model",
			Sonnet:    "sonnet-model",
			Haiku:     "haiku-model",
			SmallFast: "small-model",
		},
	}
}

func TestBuildEnvOnAnthropicClearsGatewayLeftovers(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"ANTHROPIC_BASE_URL=https://api.akashml.com/anthropic",
		"ANTHROPIC_AUTH_TOKEN=stale-token",
		"ANTHROPIC_DEFAULT_SONNET_MODEL=zai-org/GLM-5.3",
		"ANTHROPIC_SMALL_FAST_MODEL=zai-org/GLM-5.3",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1",
	}

	got := envMap(BuildEnv(base, ProviderAnthropic, Provider{}, ""))

	for _, name := range managedVars {
		if name == "CX_PROVIDER" {
			continue
		}
		if value, ok := got[name]; ok {
			t.Errorf("%s survived as %q, it must be cleared so the subscription is used", name, value)
		}
	}
	if got["PATH"] != "/usr/bin" {
		t.Errorf("PATH = %q, unrelated variables must survive", got["PATH"])
	}
	if got["CX_PROVIDER"] != ProviderAnthropic {
		t.Errorf("CX_PROVIDER = %q, want %q", got["CX_PROVIDER"], ProviderAnthropic)
	}
}

func TestBuildEnvOnAnthropicKeepsCloudProviderSelection(t *testing.T) {
	got := envMap(BuildEnv([]string{"CLAUDE_CODE_USE_BEDROCK=1"}, ProviderAnthropic, Provider{}, ""))

	if got["CLAUDE_CODE_USE_BEDROCK"] != "1" {
		t.Error("CLAUDE_CODE_USE_BEDROCK was cleared, but a Bedrock user's own setup must be left alone")
	}
}

func TestBuildEnvOnGatewaySetsEveryModelTier(t *testing.T) {
	got := envMap(BuildEnv([]string{"PATH=/usr/bin"}, ProviderAkashML, testProvider(), "secret-token"))

	want := map[string]string{
		"ANTHROPIC_BASE_URL":                     "https://api.akashml.com/anthropic",
		"ANTHROPIC_AUTH_TOKEN":                   "secret-token",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":           "opus-model",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":         "sonnet-model",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":          "haiku-model",
		"ANTHROPIC_SMALL_FAST_MODEL":             "small-model",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
		"CX_PROVIDER":                            ProviderAkashML,
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("%s = %q, want %q", name, got[name], value)
		}
	}
}

func TestBuildEnvOnGatewayClearsHigherPrecedenceCredentials(t *testing.T) {
	base := []string{
		"ANTHROPIC_API_KEY=sk-leftover",
		"CLAUDE_CODE_USE_BEDROCK=1",
		"CLAUDE_CODE_USE_VERTEX=1",
	}

	got := envMap(BuildEnv(base, ProviderAkashML, testProvider(), "secret-token"))

	for _, name := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX"} {
		if _, ok := got[name]; ok {
			t.Errorf("%s survived, it outranks ANTHROPIC_AUTH_TOKEN and would bypass the gateway", name)
		}
	}
}

func TestBuildEnvNeverDuplicatesAVariable(t *testing.T) {
	base := []string{"ANTHROPIC_BASE_URL=https://api.anthropic.com", "PATH=/usr/bin"}

	seen := map[string]int{}
	for _, entry := range BuildEnv(base, ProviderAkashML, testProvider(), "secret-token") {
		name, _, _ := strings.Cut(entry, "=")
		seen[name]++
	}

	for name, count := range seen {
		if count > 1 {
			t.Errorf("%s appears %d times, a duplicate makes the winning value depend on the reader", name, count)
		}
	}
}

func TestResolveRejectsAnUnconfiguredProvider(t *testing.T) {
	if _, err := DefaultConfig().Resolve("nope"); err == nil {
		t.Fatal("Resolve accepted a provider that is not in the config")
	}
	if _, err := DefaultConfig().Resolve(ProviderAnthropic); err != nil {
		t.Fatalf("Resolve(anthropic) returned %v, anthropic needs no config entry", err)
	}
}

func envMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if ok {
			out[name] = value
		}
	}
	return out
}

func TestBuildEnvOnGatewayDisablesTheLongContextVariant(t *testing.T) {
	got := envMap(BuildEnv(nil, ProviderAkashML, testProvider(), "secret-token"))

	// A saved 1M selection otherwise makes Claude Code request "<model>[1m]",
	// which no gateway model id carries.
	if got["CLAUDE_CODE_DISABLE_1M_CONTEXT"] != "1" {
		t.Error("CLAUDE_CODE_DISABLE_1M_CONTEXT was not set, so a 1M model selection would be sent as an unknown model id")
	}
}

func TestBuildEnvOnAnthropicLeavesTheLongContextVariantAlone(t *testing.T) {
	got := envMap(BuildEnv(nil, ProviderAnthropic, Provider{}, ""))

	if _, ok := got["CLAUDE_CODE_DISABLE_1M_CONTEXT"]; ok {
		t.Error("CLAUDE_CODE_DISABLE_1M_CONTEXT was set on the Anthropic path, where 1M context is wanted")
	}
}
