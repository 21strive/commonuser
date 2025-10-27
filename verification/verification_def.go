package verification

import "errors"

var VerificationNotFound = errors.New("Verification not found")
var InvalidVerificationCode = errors.New("Invalid verification code")
