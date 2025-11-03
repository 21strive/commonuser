package account

import "errors"

var NotFound = errors.New("account not found!")
var SeedRequired = errors.New("seed required!")
