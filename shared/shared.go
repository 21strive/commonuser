package shared

import (
	"errors"
	"time"
)

var InvalidToken = errors.New("invalid token")
var RequestExpired = errors.New("request expired")
var RequestNotFound = errors.New("request not found")
var Unauthorized = errors.New("unauthorized")

var BaseTTL = 24 * time.Hour
var SortedSetTTL = 12 * time.Hour
