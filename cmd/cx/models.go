package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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

	if len(args) == 0 {
		return listModels(models, p)
	}
	if args[0] != "use" {
		return fmt.Errorf("unknown models command %q, try: cx --%s models use <number|id> [tier...]", args[0], provider)
	}
	return useModel(cfg, provider, configPath, models, args[1:])
}

func listModels(models []cx.Model, p cx.Provider) error {
	out := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, m := range models {
		fmt.Fprintf(out, "%d\t%s\t%s\t%s\n", i+1, m.ID, m.Name(), strings.Join(p.Models.TiersUsing(m.ID), ", "))
	}
	if err := out.Flush(); err != nil {
		return err
	}
	fmt.Println("\nuse one everywhere:  cx --akash models use <number>")
	fmt.Println("or for one tier:     cx --akash models use <number> haiku")
	return nil
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
	cfg.Providers[provider] = p

	if err := cx.Save(configPath, cfg); err != nil {
		return err
	}

	fmt.Printf("%s now backs %s\n", chosen.ID, strings.Join(tiers, ", "))
	for _, tier := range cx.Tiers {
		fmt.Printf("  %-10s %s\n", tier, p.Models.Get(tier))
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
