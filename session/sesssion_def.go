package session

import "errors"

var SessionNotFound = errors.New("session not found")
var InvalidSession = errors.New("invalid session")
var SeedRequired = errors.New("seed required")
