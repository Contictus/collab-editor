package sync

import "errors"

var (
	errMissingCredentials = errors.New("sync: missing doc or token")
	errForbidden          = errors.New("sync: no document access")
)
