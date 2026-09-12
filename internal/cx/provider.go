package cx

import (
	"fmt"
	"sort"
	"strings"
)

// managedVars are cleared before every launch, whichever provider is chosen, so
// that a token exported by an earlier gateway session can never leak into a run
// that is meant to use the Anthropic subscription.
var managedVars = []string{
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
	"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
	"CLAUDE_CODE_DISABLE_1M_CONTEXT",
	"CX_PROVIDER",
}

// cloudProviderVars outrank ANTHROPIC_AUTH_TOKEN in Claude Code's auth
// precedence, so a gateway run has to clear them. A run on Anthropic proper
// leaves them alone, since that is how a Bedrock or Vertex user is set up.
var cloudProviderVars = []string{
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
	"CLAUDE_CODE_USE_FOUNDRY",
	"CLAUDE_CODE_USE_ANTHROPIC_AWS",
}

// BuildEnv returns the environment to launch Claude Code with. token is used
// only by gateway providers and is ignored for ProviderAnthropic.
func BuildEnv(base []string, name string, p Provider, token string) []string {
	drop := append([]string{}, managedVars...)
	if name != ProviderAnthropic {
		drop = append(drop, cloudProviderVars...)
	}

	env := strip(base, drop)
	env = append(env, "CX_PROVIDER="+name)
	if name == ProviderAnthropic {
		return env
	}

	for k, v := range gatewayVars(p, token) {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

func gatewayVars(p Provider, token string) map[string]string {
	return map[string]string{
		"ANTHROPIC_BASE_URL":                     p.BaseURL,
		"ANTHROPIC_AUTH_TOKEN":                   token,
		"ANTHROPIC_DEFAULT_OPUS_MODEL":           p.Models.Opus,
		"ANTHROPIC_DEFAULT_SONNET_MODEL":         p.Models.Sonnet,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":          p.Models.Haiku,
		"ANTHROPIC_SMALL_FAST_MODEL":             p.Models.SmallFast,
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
		// A saved 1M-context model selection carries into a gateway session and
		// makes Claude Code ask for "<model>[1m]", which no gateway model id is.
		"CLAUDE_CODE_DISABLE_1M_CONTEXT": "1",
	}
}

func strip(env []string, names []string) []string {
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}

	kept := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, ok := strings.Cut(entry, "=")
		if ok && drop[name] {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

func (c Config) Resolve(name string) (Provider, error) {
	if name == ProviderAnthropic {
		return Provider{}, nil
	}
	p, ok := c.Providers[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q: add it to your config or use --anthropic", name)
	}
	if p.BaseURL == "" {
		return Provider{}, fmt.Errorf("provider %q has no baseUrl", name)
	}
	return p, nil
}
