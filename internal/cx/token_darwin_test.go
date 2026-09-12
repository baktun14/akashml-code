//go:build darwin

package cx

import (
	"strings"
	"testing"
)

func TestSecurityStoreCommandKeepsTheSecretOutOfArgv(t *testing.T) {
	const secret = "akml-EXAMPLE-do-not-put-me-in-the-process-table"

	cmd := securityStoreCommand("svc", "acct", secret)

	for _, arg := range cmd.Args {
		if strings.Contains(arg, secret) {
			t.Fatal("the token was passed as an argument, where any process on the machine can read it from ps")
		}
	}
}

func TestSecurityStoreCommandDetachesFromTheControllingTerminal(t *testing.T) {
	cmd := securityStoreCommand("svc", "acct", "akml-EXAMPLE-token")

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatal("without Setsid, security reads the secret from /dev/tty and stores the terminal's keystrokes instead of the token")
	}
}
