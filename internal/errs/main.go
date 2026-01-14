package errs

import (
	"errors"
)

var AccountNotFound = errors.New("account not found!")
var AccountSeedRequired = errors.New("account seed required")

var RequestNotFound = errors.New("request not found")
