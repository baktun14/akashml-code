package cx

import "errors"

// ErrNoToken means no credential has been stored yet, as opposed to a store
// that could not be read at all.
var ErrNoToken = errors.New("no token stored")
