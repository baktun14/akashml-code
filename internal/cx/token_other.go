//go:build !darwin

package cx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func tokenPath(service string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, service+".token"), nil
}

func LoadToken(service string) (string, error) {
	path, err := tokenPath(service)
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", ErrNoToken
	}
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func StoreToken(service, _, token string) error {
	path, err := tokenPath(service)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}

func TokenLocation(service string) string {
	path, err := tokenPath(service)
	if err != nil {
		return service + ".token"
	}
	return path
}
