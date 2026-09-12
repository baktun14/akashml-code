package cx

import (
	"fmt"
	"strings"
)

// CheckToken rejects a credential that cannot be this provider's, so a mistyped
// or mispasted secret is caught at the prompt instead of surfacing later as an
// opaque 401 from the gateway.
func CheckToken(p Provider, token string) error {
	switch {
	case token == "":
		return fmt.Errorf("the token is empty")
	case p.TokenPrefix == "":
		return nil
	case !strings.HasPrefix(token, p.TokenPrefix):
		return fmt.Errorf("this does not look like a key for this provider: its keys start with %q", p.TokenPrefix)
	}
	return nil
}

// MaskToken renders a credential for display, showing only enough to tell two
// keys apart.
func MaskToken(token string) string {
	const shown = 3
	if len(token) < 2*shown+2 {
		return fmt.Sprintf("%d characters", len(token))
	}
	return fmt.Sprintf("%s...%s (%d characters)", token[:shown], token[len(token)-shown:], len(token))
}

// StoreToken writes a credential and then reads it back before reporting
// success. security(1) has been seen to exit 0 on an update that moved the
// item's modification date but left the old password in place, so its exit code
// alone is not evidence that the write took.
func StoreToken(service, account, token string) error {
	if err := storeToken(service, account, token); err != nil {
		return err
	}

	stored, err := LoadToken(service)
	if err != nil {
		return fmt.Errorf("wrote the token but could not read it back: %w", err)
	}
	if stored != token {
		return fmt.Errorf(
			"the store reported success but %s still holds a different value; remove it and try again:\n    security delete-generic-password -s %s",
			TokenLocation(service), service)
	}
	return nil
}
