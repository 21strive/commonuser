package database

type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
