package cx

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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
	"API_TIMEOUT_MS",
	"CLAUDE_CODE_MAX_CONTEXT_TOKENS",
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
	vars := map[string]string{
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
	if p.APITimeoutMS > 0 {
		vars["API_TIMEOUT_MS"] = strconv.Itoa(p.APITimeoutMS)
	}
	if p.MaxContextTokens > 0 {
		vars["CLAUDE_CODE_MAX_CONTEXT_TOKENS"] = strconv.Itoa(p.MaxContextTokens)
	}
	return vars
}

type modelPickerRow struct {
	Model     string `json:"model"`
	Label     string `json:"label,omitempty"`
	BehavesAs string `json:"behavesAs"`
}

// ModelPickerSettings describes this provider's models to Claude Code, which
// otherwise refuses any id missing from the catalog its own build shipped with.
// It is passed per launch via --settings so that a gateway's models never enter
// the picker of a session running on Anthropic.
func ModelPickerSettings(name string, p Provider) (string, error) {
	if p.BehavesAs == "" {
		return "", nil
	}

	rows := make([]modelPickerRow, 0, 4)
	for _, model := range p.Models.distinct() {
		rows = append(rows, modelPickerRow{
			Model:     model,
			Label:     fmt.Sprintf("%s (%s)", model, name),
			BehavesAs: p.BehavesAs,
		})
	}
	if len(rows) == 0 {
		return "", nil
	}

	payload, err := json.Marshal(map[string]any{
		"modelPicker": map[string]any{"options": rows},
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// distinct lists the configured model ids once each, in tier order.
func (m Models) distinct() []string {
	seen := map[string]bool{}
	var out []string
	for _, model := range []string{m.Opus, m.Sonnet, m.Haiku, m.SmallFast} {
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		out = append(out, model)
	}
	return out
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
