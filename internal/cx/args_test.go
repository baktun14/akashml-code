package cx

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want Invocation
	}{
		{
			name: "no arguments launches the default provider",
			argv: nil,
			want: Invocation{},
		},
		{
			name: "provider flag alone",
			argv: []string{"--akash"},
			want: Invocation{Provider: ProviderAkashML},
		},
		{
			name: "provider flag then claude flags",
			argv: []string{"--akash", "--continue"},
			want: Invocation{Provider: ProviderAkashML, ClaudeArgs: []string{"--continue"}},
		},
		{
			name: "claude flags with no provider flag",
			argv: []string{"--continue"},
			want: Invocation{ClaudeArgs: []string{"--continue"}},
		},
		{
			name: "aliases select the same providers",
			argv: []string{"--claude"},
			want: Invocation{Provider: ProviderAnthropic},
		},
		{
			name: "named provider",
			argv: []string{"--provider=someGateway", "-p", "hi"},
			want: Invocation{Provider: "someGateway", ClaudeArgs: []string{"-p", "hi"}},
		},
		{
			name: "the last provider flag wins",
			argv: []string{"--anthropic", "--akash"},
			want: Invocation{Provider: ProviderAkashML},
		},
		{
			name: "subcommand",
			argv: []string{"status"},
			want: Invocation{Command: CommandStatus},
		},
		{
			name: "subcommand after a provider flag",
			argv: []string{"--akash", "token", "set"},
			want: Invocation{Provider: ProviderAkashML, Command: CommandToken, CommandArgs: []string{"set"}},
		},
		{
			name: "help flags map to the help command",
			argv: []string{"-h"},
			want: Invocation{Command: CommandHelp},
		},
		{
			name: "double dash forces a subcommand word through to claude",
			argv: []string{"--", "status"},
			want: Invocation{ClaudeArgs: []string{"status"}},
		},
		{
			name: "double dash forces a provider flag through to claude",
			argv: []string{"--", "--akash"},
			want: Invocation{ClaudeArgs: []string{"--akash"}},
		},
		{
			name: "a provider flag after a claude flag belongs to claude",
			argv: []string{"-p", "--akash"},
			want: Invocation{ClaudeArgs: []string{"-p", "--akash"}},
		},
		{
			name: "an unknown flag is claude's, not an error",
			argv: []string{"--some-future-claude-flag"},
			want: Invocation{ClaudeArgs: []string{"--some-future-claude-flag"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseArgs(tt.argv)

			if got.Provider != tt.want.Provider {
				t.Errorf("Provider = %q, want %q", got.Provider, tt.want.Provider)
			}
			if got.Command != tt.want.Command {
				t.Errorf("Command = %q, want %q", got.Command, tt.want.Command)
			}
			if !sameArgs(got.CommandArgs, tt.want.CommandArgs) {
				t.Errorf("CommandArgs = %v, want %v", got.CommandArgs, tt.want.CommandArgs)
			}
			if !sameArgs(got.ClaudeArgs, tt.want.ClaudeArgs) {
				t.Errorf("ClaudeArgs = %v, want %v", got.ClaudeArgs, tt.want.ClaudeArgs)
			}
		})
	}
}

func sameArgs(got, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}
