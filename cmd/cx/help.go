package main

import (
	"fmt"
	"os"
)

const usage = `cx launches Claude Code against AkashML or against Anthropic.

usage:
  cx [provider flag] [claude args...]
  cx [provider flag] <command>

provider flags:
  --akash, --akashml     run this session on AkashML
  --anthropic, --claude  run this session on Anthropic
  --provider=NAME        run this session on NAME from your config
  --                     stop reading cx flags, pass the rest to claude

commands:
  status                 show the resolved provider, models and token state
  models                 list the models the gateway serves
  models use N [tier]    point every tier, or one tier, at that model
  token set              read a token from the terminal and store it securely
  help, version

Anything cx does not recognise goes to claude untouched, so cx --akash
--continue resumes the current conversation on AkashML. Claude Code's own
help is one level down: cx -- --help.

Models and endpoints live in the config file that cx status points at.
`

func printHelp() error {
	_, err := fmt.Fprint(os.Stdout, usage)
	return err
}
