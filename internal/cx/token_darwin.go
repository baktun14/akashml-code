//go:build darwin

package cx

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

func storeToken(service, account, token string) error {
	cmd := exec.Command(securityBin, "add-generic-password", "-U", "-s", service, "-a", account, "-w")
	// security prompts for the secret and then for a confirmation. Feeding both
	// on stdin keeps the token out of the process table, where passing it as an
	// argument would leave it readable by any other process on the machine.
	cmd.Stdin = strings.NewReader(token + "\n" + token + "\n")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storing %q in the keychain: %w: %s", service, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func TokenLocation(service string) string {
	return fmt.Sprintf("macOS keychain, service %q", service)
}
