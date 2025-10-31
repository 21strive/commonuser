package update_email

import "errors"

var TicketNotFound = errors.New("Update email ticket not found")
var InvalidToken = errors.New("Invalid token")
