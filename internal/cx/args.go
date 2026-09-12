package cx

import "strings"

const (
	CommandLaunch  = ""
	CommandStatus  = "status"
	CommandToken   = "token"
	CommandHelp    = "help"
	CommandVersion = "version"
)

type Invocation struct {
	// Provider is empty when no flag selected one, meaning the configured default.
	Provider    string
	Command     string
	CommandArgs []string
	ClaudeArgs  []string
}

// ParseArgs consumes the leading flags cx owns and hands everything after them
// to Claude Code untouched, so an unrecognised flag is Claude's rather than an
// error. "--" ends cx's own parsing early and forces the rest through.
func ParseArgs(argv []string) Invocation {
	inv := Invocation{}
	i := 0
	passThroughOnly := false

	for i < len(argv) {
		arg := argv[i]
		if arg == "--" {
			passThroughOnly = true
			i++
			break
		}

		provider, ok := providerFor(arg)
		if !ok {
			break
		}
		inv.Provider = provider
		i++
	}

	rest := argv[i:]
	if !passThroughOnly && len(rest) > 0 {
		if cmd, ok := commandFor(rest[0]); ok {
			inv.Command = cmd
			inv.CommandArgs = rest[1:]
			return inv
		}
	}

	inv.ClaudeArgs = rest
	return inv
}

func providerFor(arg string) (string, bool) {
	switch arg {
	case "--akash", "--akashml":
		return ProviderAkashML, true
	case "--anthropic", "--claude":
		return ProviderAnthropic, true
	}
	if name, ok := strings.CutPrefix(arg, "--provider="); ok {
		return name, true
	}
	return "", false
}

func commandFor(arg string) (string, bool) {
	switch arg {
	case "status":
		return CommandStatus, true
	case "token":
		return CommandToken, true
	case "help", "--help", "-h":
		return CommandHelp, true
	case "version", "--version":
		return CommandVersion, true
	}
	return "", false
}
