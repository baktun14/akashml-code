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

It reads the token from your terminal without echoing it and writes it to your keychain, then reads it back to confirm the write actually took. Nothing touches your shell history, and the token is never passed as a command-line argument, so it never shows up in `ps`.

If it ever reports that the store still holds a different value, delete the item and run it again:

```sh
security delete-generic-password -s akashml
```

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
  opus        zai-org--GLM-5.3
  sonnet      zai-org--GLM-5.3
  haiku       zai-org--GLM-5.3
  small/fast  zai-org--GLM-5.3
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
        "opus": "zai-org--GLM-5.3",
        "sonnet": "zai-org--GLM-5.3",
        "haiku": "zai-org--GLM-5.3",
        "smallFast": "zai-org--GLM-5.3"
      },
      "behavesAs": "claude-sonnet-5",
      "apiTimeoutMs": 3000000
    }
  }
}
```

Model ids use `--` between the org and the model, not `/`. Ask the endpoint for the current list rather than guessing:

```sh
curl -s https://api.akashml.com/anthropic/v1/models | python3 -m json.tool
```

Trust that list over any documentation, including AkashML's own guide, which currently advertises at least one model the endpoint answers with a 404. The gateway accepts both the `zai-org--GLM-5.3` spelling the endpoint returns and the `zai-org/GLM-5.3` spelling the docs use.

`smallFast` is worth setting separately. Claude Code uses that tier for frequent background work, so a cheaper model there, `openai--gpt-oss-20b` for instance, is rarely something you notice.

Set `"defaultProvider": "akashml"` if you would rather have a bare `cx` go to AkashML and keep `cx --anthropic` for the times you want your subscription.

`behavesAs` names the Claude model whose capabilities Claude Code should assume for this provider's models, and it is not optional in practice. Claude Code refuses any model id missing from the catalog its own build shipped with, so without it a launch dies on `[claude-code:unrecognized_model]` before a single request goes out. `cx` passes it per launch via `claude --settings`, so a gateway's models never show up in the picker of a session running on Anthropic.

`apiTimeoutMs` raises the per-request deadline. Open models behind a gateway can be far slower than Claude is: with Claude Code's full system prompt and tool definitions in the request, GLM-5.3 has taken several minutes to answer a one-word prompt. The default here is the 3000000 ms AkashML's own guide recommends, which is 50 minutes. That is deliberately generous for long agent turns, and the tradeoff is that a genuinely stuck request takes 50 minutes to give up rather than failing fast. Lower it if you would rather find out sooner.

`tokenPrefix` is what that provider's keys begin with. `cx` checks it when you store a key and again before launching, so a mistyped credential fails at the prompt with a readable message instead of reaching you as a 401 from the gateway.

Any gateway that speaks the Anthropic API works. Add it under `providers` with its own `keychainService`, then run `cx --provider=NAME token set`.

## What it sets

On the Anthropic path, `cx` clears the variables a gateway session would have set and launches. Your saved `claude` login is used exactly as it would be normally.

On a gateway path, it clears the same set and then exports the base URL, your token as `ANTHROPIC_AUTH_TOKEN`, all four model tiers, and two flags. `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1` keeps Claude Code from sending pre-release features a third-party endpoint won't implement. `CLAUDE_CODE_DISABLE_1M_CONTEXT=1` matters more than it looks: if your saved model selection is a 1M-context variant, Claude Code carries that choice into the gateway session and asks for `<model>[1m]`, an id no gateway serves, and you get `unrecognized_model` before a single request goes out. It also clears `CLAUDE_CODE_USE_BEDROCK` and friends, because those outrank the auth token in [Claude Code's precedence order](https://code.claude.com/docs/en/authentication#authentication-precedence) and would route you somewhere you didn't ask for.

One thing to avoid: don't put `ANTHROPIC_*` variables in the `env` block of `~/.claude/settings.json`. Values there override the environment a process is launched with, which would pin every session to one provider and defeat the tool. AkashML's own Claude Code guide tells you to do exactly that, which is fine if the gateway is all you ever use and is the thing `cx` exists to replace.

## Caveats

Resuming a conversation across providers works, but Claude Code has no documented position on it, and the transcript carries the model ids it was recorded with. In practice `cx --akash --continue` picks up where you left off.

Keychain storage is macOS only. On other platforms the token goes to a `0600` file next to the config instead.

## License

MIT
