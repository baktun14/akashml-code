# akashml-code

`cx` launches Claude Code against [AkashML](https://akashml.com) or against Anthropic, picking the provider per command instead of per shell.

The usual way to point Claude Code at a gateway is to export `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` from a shell function. That leaves the token sitting in your dotfiles in plaintext, and once you run the function, every later `claude` in that terminal quietly keeps using the gateway. `cx` fixes both. Your token lives in the macOS keychain, and each launch starts from a clean environment.

```
cx                      # Claude Code on your Anthropic subscription
cx --akash              # Claude Code on AkashML
cx --akash --continue   # resume the conversation you were just in, on AkashML
```

That last one is the point of the tool. When you hit your usage limit mid-task, Ctrl-C and run it, and your session carries on.

## Install

Go 1.22 or newer:

```sh
go install github.com/baktun14/akashml-code/cmd/cx@latest
```

Or from a clone:

```sh
go build -o ~/.local/bin/cx ./cmd/cx
```

Then store your AkashML token, which you can generate at [akashml.com](https://akashml.com):

```sh
cx token set
```

It reads the token from your terminal without echoing it and writes it to your keychain. Nothing touches your shell history, and the token is never passed as a command-line argument, so it never shows up in `ps`.

Check where you stand at any time:

```
$ cx status
provider  anthropic
default   anthropic
config    /Users/you/.config/cx/config.json
claude    /Users/you/.local/bin/claude

gateway       akashml
  base url    https://api.akashml.com/anthropic
  token       present (macOS keychain, service "akashml")
  opus        zai-org/GLM-5.3
  sonnet      zai-org/GLM-5.3
  haiku       zai-org/GLM-5.3
  small/fast  zai-org/GLM-5.3
```

## Usage

| | |
|---|---|
| `--akash`, `--akashml` | run this session on AkashML |
| `--anthropic`, `--claude` | run this session on Anthropic |
| `--provider=NAME` | run this session on a gateway from your config |
| `--` | stop reading `cx` flags, pass everything after it to `claude` |
| `status` | show the resolved provider, models and token state |
| `token set` | read a token from the terminal and store it |

Anything `cx` doesn't recognise goes to `claude` untouched, so every Claude Code flag keeps working: `cx --akash --resume`, `cx -p "explain this"`, `cx --akash mcp list`. Claude Code's own help is one level down, at `cx -- --help`.

Since `cx` replaces itself with `claude` rather than wrapping it, exit codes, signals and terminal handling behave exactly as they would if you had typed `claude`.

## Configuring models

`cx status` prints the path to `config.json`. Edit it to change which AkashML model backs each of Claude Code's tiers:

```json
{
  "defaultProvider": "anthropic",
  "providers": {
    "akashml": {
      "baseUrl": "https://api.akashml.com/anthropic",
      "keychainService": "akashml",
      "models": {
        "opus": "zai-org/GLM-5.3",
        "sonnet": "zai-org/GLM-5.3",
        "haiku": "zai-org/GLM-5.3",
        "smallFast": "zai-org/GLM-5.3"
      }
    }
  }
}
```

`smallFast` is worth setting separately. Claude Code uses that tier for frequent background work, so a cheaper model there is rarely something you notice.

Set `"defaultProvider": "akashml"` if you would rather have a bare `cx` go to AkashML and keep `cx --anthropic` for the times you want your subscription.

Any gateway that speaks the Anthropic API works. Add it under `providers` with its own `keychainService`, then run `cx --provider=NAME token set`.

## What it sets

On the Anthropic path, `cx` clears the variables a gateway session would have set and launches. Your saved `claude` login is used exactly as it would be normally.

On a gateway path, it clears the same set and then exports the base URL, your token as `ANTHROPIC_AUTH_TOKEN`, all four model tiers, and `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1`, which keeps Claude Code from sending pre-release features a third-party endpoint won't implement. It also clears `CLAUDE_CODE_USE_BEDROCK` and friends, because those outrank the auth token in [Claude Code's precedence order](https://code.claude.com/docs/en/authentication#authentication-precedence) and would route you somewhere you didn't ask for.

One thing to avoid: don't put `ANTHROPIC_*` variables in the `env` block of `~/.claude/settings.json`. Values there override the environment a process is launched with, which would pin every session to one provider and defeat the tool.

## Caveats

Resuming a conversation across providers works, but Claude Code has no documented position on it, and the transcript carries the model ids it was recorded with. In practice `cx --akash --continue` picks up where you left off.

Keychain storage is macOS only. On other platforms the token goes to a `0600` file next to the config instead.

## License

MIT
