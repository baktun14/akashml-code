package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/baktun14/akashml-code/internal/cx"
)

func runModels(cfg cx.Config, provider, configPath string, args []string) error {
	if provider == cx.ProviderAnthropic {
		return errors.New("model switching applies to a gateway, try: cx --akash models")
	}

	p, err := cfg.Resolve(provider)
	if err != nil {
		return err
	}

	token, err := cx.LoadToken(p.KeychainService)
	if err != nil && !errors.Is(err, cx.ErrNoToken) {
		return err
	}

	models, err := cx.FetchModels(p, token)
	if err != nil {
		return err
	}

	switch {
	case len(args) == 0:
		return listModels(models, provider, p)
	case args[0] == "use":
		return useModel(cfg, provider, configPath, models, args[1:])
	case args[0] == "probe":
		return probeModels(models, provider, p, token)
	}
	return fmt.Errorf("unknown models command %q, try: use or probe", args[0])
}

func listModels(models []cx.Model, provider string, p cx.Provider) error {
	caps := loadCapabilities()

	out := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	probed := false
	for i, m := range models {
		vision := caps.VisionFor(provider, m.ID)
		probed = probed || vision != cx.VisionUnknown
		fmt.Fprintf(out, "%d\t%s\t%s\t%s\t%s\t%s\n",
			i+1, m.ID, m.Name(), describeContext(m), strings.Join(p.Models.TiersUsing(m.ID), ", "), vision.Describe())
	}
	if err := out.Flush(); err != nil {
		return err
	}

	fmt.Printf("\nuse one everywhere:  cx --%s models use <number>\n", provider)
	fmt.Printf("or for one tier:     cx --%s models use <number> haiku\n", provider)
	if !probed {
		fmt.Printf("which take images:   cx --%s models probe\n", provider)
	}
	return nil
}

// probeModels sends each model an image and records what it does with it, since
// the model list says nothing about capabilities and the failure mode for
// getting this wrong is a session that cannot continue.
func probeModels(models []cx.Model, provider string, p cx.Provider, token string) error {
	fmt.Fprintf(os.Stderr, "sending each of the %d models one small image...\n", len(models))

	found := make([]cx.VisionSupport, len(models))
	errs := make([]error, len(models))

	var wg sync.WaitGroup
	limit := make(chan struct{}, 4)
	for i, m := range models {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			found[i], errs[i] = cx.ProbeVision(p, token, m.ID)
		}()
	}
	wg.Wait()

	caps := loadCapabilities()
	out := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, m := range models {
		if errs[i] != nil {
			fmt.Fprintf(out, "%s\t%s\n", m.ID, "could not tell: "+errs[i].Error())
			continue
		}
		caps.SetVision(provider, m.ID, found[i])
		fmt.Fprintf(out, "%s\t%s\n", m.ID, found[i].Describe())
	}
	if err := out.Flush(); err != nil {
		return err
	}

	path, err := cx.CapabilitiesPath()
	if err != nil {
		return err
	}
	if err := cx.SaveCapabilities(path, caps); err != nil {
		return err
	}
	fmt.Println("\nA model that drops images gives no error, it just answers as if the image were not there.")
	return nil
}

// describeContext renders a window in the units people quote them in.
func describeContext(m cx.Model) string {
	switch {
	case m.ContextLength == 0:
		return ""
	case m.ContextLength >= 1<<20:
		return fmt.Sprintf("%gM ctx", float64(m.ContextLength)/(1<<20))
	default:
		return fmt.Sprintf("%gk ctx", float64(m.ContextLength)/(1<<10))
	}
}

func loadCapabilities() cx.Capabilities {
	path, err := cx.CapabilitiesPath()
	if err != nil {
		return cx.Capabilities{}
	}
	return cx.LoadCapabilities(path)
}

func useModel(cfg cx.Config, provider, configPath string, models []cx.Model, args []string) error {
	if len(args) == 0 {
		return errors.New("say which model, by number or id: cx --akash models use <number|id> [tier...]")
	}

	chosen, err := pickModel(models, args[0])
	if err != nil {
		return err
	}

	tiers := cx.Tiers
	if len(args) > 1 {
		tiers = nil
		for _, name := range args[1:] {
			tier, ok := cx.TierNamed(name)
			if !ok {
				return fmt.Errorf("unknown tier %q, expected one of: opus, sonnet, haiku, smallFast", name)
			}
			tiers = append(tiers, tier)
		}
	}

	p := cfg.Providers[provider]
	for _, tier := range tiers {
		p.Models.Set(tier, chosen.ID)
	}
	p.MaxContextTokens = cx.ConversationWindow(p.Models, models)
	cfg.Providers[provider] = p

	if err := cx.Save(configPath, cfg); err != nil {
		return err
	}

	if vision := loadCapabilities().VisionFor(provider, chosen.ID); vision == cx.VisionRejects {
		fmt.Fprintf(os.Stderr,
			"warning: %s rejects images. Pasting one into a session on it fails every turn afterwards,\n"+
				"         including the message asking it to ignore the image.\n", chosen.ID)
	}

	fmt.Printf("%s now backs %s\n", chosen.ID, strings.Join(tiers, ", "))
	for _, tier := range cx.Tiers {
		fmt.Printf("  %-10s %s\n", tier, p.Models.Get(tier))
	}
	if p.MaxContextTokens > 0 {
		fmt.Printf("  %-10s %d tokens\n", "context", p.MaxContextTokens)
	}
	return nil
}

// pickModel accepts either the number shown in the listing or the id itself, so
// a person can copy an id from elsewhere without counting rows.
func pickModel(models []cx.Model, choice string) (cx.Model, error) {
	if n, err := strconv.Atoi(choice); err == nil {
		if n < 1 || n > len(models) {
			return cx.Model{}, fmt.Errorf("there is no model %d, the list has %d", n, len(models))
		}
		return models[n-1], nil
	}

	for _, m := range models {
		if m.ID == choice {
			return m, nil
		}
	}
	return cx.Model{}, fmt.Errorf("%q is not a model this provider serves, run: cx --akash models", choice)
}
