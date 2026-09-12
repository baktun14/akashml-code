package cx

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Capabilities caches what probing found, so a listing can show it without
// paying for a round of requests every time.
type Capabilities struct {
	Vision map[string]map[string]VisionSupport `json:"vision"`
}

func CapabilitiesPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "capabilities.json"), nil
}

// LoadCapabilities returns an empty set when nothing has been probed yet or the
// cache is unreadable, since a stale cache must never block a launch.
func LoadCapabilities(path string) Capabilities {
	caps := Capabilities{Vision: map[string]map[string]VisionSupport{}}

	raw, err := os.ReadFile(path)
	if err != nil {
		return caps
	}
	if err := json.Unmarshal(raw, &caps); err != nil || caps.Vision == nil {
		return Capabilities{Vision: map[string]map[string]VisionSupport{}}
	}
	return caps
}

func (c Capabilities) VisionFor(provider, model string) VisionSupport {
	return c.Vision[provider][model]
}

func (c *Capabilities) SetVision(provider, model string, support VisionSupport) {
	if c.Vision == nil {
		c.Vision = map[string]map[string]VisionSupport{}
	}
	if c.Vision[provider] == nil {
		c.Vision[provider] = map[string]VisionSupport{}
	}
	c.Vision[provider][model] = support
}

func SaveCapabilities(path string, caps Capabilities) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(caps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
