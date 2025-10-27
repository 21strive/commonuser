package shared

import (
	"database/sql"
	"errors"
)

var InvalidToken = errors.New("invalid token")
var RequestExpired = errors.New("request expired")
var RequestNotFound = errors.New("request not found")
var Unauthorized = errors.New("unauthorized")

type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
