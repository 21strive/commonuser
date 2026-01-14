package errs

import (
	"errors"
)

var AccountNotFound = errors.New("account not found!")
var AccountSeedRequired = errors.New("account seed required!")
var InvalidToken = errors.New("invalid token")
var RequestExpired = errors.New("request expired")
var RequestNotFound = errors.New("request not found")
var Unauthorized = errors.New("unauthorized")
var ResetPasswordTicketNotFound = errors.New("Reset password ticket not found")
var SessionNotFound = errors.New("session not found")
var InvalidSession = errors.New("invalid session")
var SeedRequired = errors.New("seed required")
var UpdateEmailTicketNotFound = errors.New("Update email ticket not found")
var InvalidUpdateEmailToken = errors.New("Invalid token")
