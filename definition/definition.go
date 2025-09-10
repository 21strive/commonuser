package definition

import (
	"errors"
	"time"
)

// for UpdateEmail & ResetPassword usage
var RequestExist = errors.New("request exist")
var RequestNotFound = errors.New("request not found")
var InvalidToken = errors.New("invalid token")
var RequestExpired = errors.New("request expired")
var Unauthorized = errors.New("unauthorized")

var BaseTTL = 24 * time.Hour
var SortedSetTTL = 12 * time.Hour
