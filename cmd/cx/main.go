// Command cx launches Claude Code against AkashML or against Anthropic,
// choosing the provider per invocation instead of per shell.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"slices"
	"syscall"
	"text/tabwriter"

	"github.com/baktun14/akashml-code/internal/cx"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cx: "+err.Error())
		os.Exit(1)
	}
}

func run(argv []string) error {
	inv := cx.ParseArgs(argv)

	switch inv.Command {
	case cx.CommandHelp:
		return printHelp()
	case cx.CommandVersion:
		fmt.Println("cx " + buildVersion())
		return nil
	}

	path, err := cx.ConfigPath()
	if err != nil {
		return err
	}
	cfg, err := cx.LoadConfig(path)
	if err != nil {
		return err
	}

	provider := inv.Provider
	if provider == "" {
		provider = cfg.DefaultProvider
	}

	switch inv.Command {
	case cx.CommandStatus:
		return printStatus(cfg, provider, path)
	case cx.CommandToken:
		return runToken(cfg, provider, inv.CommandArgs)
	case cx.CommandModels:
		return runModels(cfg, provider, path, inv.CommandArgs)
	}
	return launch(cfg, provider, inv.ClaudeArgs)
}

func launch(cfg cx.Config, provider string, args []string) error {
	p, err := cfg.Resolve(provider)
	if err != nil {
		return err
	}

	token, err := tokenFor(provider, p)
	if err != nil {
		return err
	}

	bin, err := claudeBin()
	if err != nil {
		return fmt.Errorf("cannot find the claude binary: %w", err)
	}

	args, err = withModelPicker(provider, p, args)
	if err != nil {
		return err
	}

	env := cx.BuildEnv(os.Environ(), provider, p, token)
	return syscall.Exec(bin, append([]string{bin}, args...), env)
}

// withModelPicker prepends the settings that describe a gateway's models, which
// Claude Code needs before it will accept an id its own catalog does not carry.
// A --settings of the caller's own is left alone rather than silently replaced.
func withModelPicker(provider string, p cx.Provider, args []string) ([]string, error) {
	if provider == cx.ProviderAnthropic {
		return args, nil
	}

	settings, err := cx.ModelPickerSettings(provider, p)
	if err != nil || settings == "" {
		return args, err
	}

	if slices.Contains(args, "--settings") {
		fmt.Fprintln(os.Stderr, "cx: --settings given, so the model picker rows for "+provider+" were not added")
		return args, nil
	}
	return append([]string{"--settings", settings}, args...), nil
}

func tokenFor(provider string, p cx.Provider) (string, error) {
	if provider == cx.ProviderAnthropic {
		return "", nil
	}

	token, err := cx.LoadToken(p.KeychainService)
	if errors.Is(err, cx.ErrNoToken) {
		return "", fmt.Errorf("no %s token stored yet, run: cx token set", provider)
	}
	if err != nil {
		return "", err
	}
	if err := cx.CheckToken(p, token); err != nil {
		return "", fmt.Errorf("the stored %s token is unusable: %w, run: cx token set", provider, err)
	}
	return token, nil
}

func claudeBin() (string, error) {
	if override := os.Getenv("CX_CLAUDE_BIN"); override != "" {
		return override, nil
	}
	return exec.LookPath("claude")
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func printStatus(cfg cx.Config, provider, configPath string) error {
	out := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(out, "provider\t%s\n", provider)
	fmt.Fprintf(out, "default\t%s\n", cfg.DefaultProvider)
	fmt.Fprintf(out, "config\t%s\n", configPath)

	if bin, err := claudeBin(); err == nil {
		fmt.Fprintf(out, "claude\t%s\n", bin)
	} else {
		fmt.Fprintf(out, "claude\tNOT FOUND on PATH\n")
	}

	for _, name := range cfg.ProviderNames() {
		p := cfg.Providers[name]
		fmt.Fprintf(out, "\ngateway\t%s\n", name)
		fmt.Fprintf(out, "  base url\t%s\n", p.BaseURL)
		fmt.Fprintf(out, "  token\t%s (%s)\n", tokenState(p), cx.TokenLocation(p.KeychainService))
		fmt.Fprintf(out, "  opus\t%s\n", p.Models.Opus)
		fmt.Fprintf(out, "  sonnet\t%s\n", p.Models.Sonnet)
		fmt.Fprintf(out, "  haiku\t%s\n", p.Models.Haiku)
		fmt.Fprintf(out, "  small/fast\t%s\n", p.Models.SmallFast)
	}
	return out.Flush()
}

func tokenState(p cx.Provider) string {
	switch token, err := cx.LoadToken(p.KeychainService); {
	case errors.Is(err, cx.ErrNoToken):
		return "missing"
	case err != nil:
		return "unreadable"
	case cx.CheckToken(p, token) != nil:
		return "present but not a valid " + p.KeychainService + " key"
	default:
		return "present, " + cx.MaskToken(token)
	}
}

func runToken(cfg cx.Config, provider string, args []string) error {
	if len(args) == 0 || args[0] != "set" {
		return errors.New("usage: cx [--provider=NAME] token set")
	}

	// Storing a token against Anthropic proper is meaningless, so an unqualified
	// "cx token set" targets the gateway even when the default provider is not.
	if provider == cx.ProviderAnthropic {
		provider = cx.ProviderAkashML
	}

	p, err := cfg.Resolve(provider)
	if err != nil {
		return err
	}

	token, err := readSecret(promptFor(provider, p))
	if err != nil {
		return err
	}
	if err := cx.CheckToken(p, token); err != nil {
		return fmt.Errorf("%w, nothing was stored", err)
	}

	if err := cx.StoreToken(p.KeychainService, os.Getenv("USER"), token); err != nil {
		return err
	}
	fmt.Printf("stored %s in %s\n", cx.MaskToken(token), cx.TokenLocation(p.KeychainService))
	return nil
}

func promptFor(provider string, p cx.Provider) string {
	if p.TokenPrefix == "" {
		return fmt.Sprintf("Paste your %s API key: ", provider)
	}
	return fmt.Sprintf("Paste your %s API key (%s...): ", provider, p.TokenPrefix)
}
