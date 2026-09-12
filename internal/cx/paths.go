package cx

import (
	"os"
	"path/filepath"
)

// ConfigDir follows the XDG convention rather than os.UserConfigDir, which on
// macOS points at ~/Library/Application Support. A CLI's config belongs
// somewhere a person can reach without quoting a path with spaces in it.
func ConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "cx"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cx"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
