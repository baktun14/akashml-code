//go:build darwin

package cx

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

const securityBin = "/usr/bin/security"

// securityItemNotFound is the exit code security(1) uses for a keychain lookup
// that matched nothing.
const securityItemNotFound = 44

func LoadToken(service string) (string, error) {
	out, err := exec.Command(securityBin, "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == securityItemNotFound {
			return "", ErrNoToken
		}
		return "", fmt.Errorf("reading %q from the keychain: %w", service, err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// securityStoreCommand builds the write without ever placing the secret in
// argv, where any other process on the machine could read it out of the process
// table.
func securityStoreCommand(service, account, token string) *exec.Cmd {
	cmd := exec.Command(securityBin, "add-generic-password", "-U", "-s", service, "-a", account, "-w")

	// Given a controlling terminal, security reads the secret from /dev/tty and
	// ignores stdin, storing whatever the terminal hands it rather than the value
	// below. Detaching the child leaves it no tty to open.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// security asks for the secret and then for a confirmation.
	cmd.Stdin = strings.NewReader(token + "\n" + token + "\n")
	return cmd
}

func storeToken(service, account, token string) error {
	if out, err := securityStoreCommand(service, account, token).CombinedOutput(); err != nil {
		return fmt.Errorf("storing %q in the keychain: %w: %s", service, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func TokenLocation(service string) string {
	return fmt.Sprintf("macOS keychain, service %q", service)
}
