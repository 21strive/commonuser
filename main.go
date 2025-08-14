package commonuser

import (
	"database/sql"
	"github.com/21strive/commonuser/lib"
	"github.com/21strive/commonuser/lib/postgresql"
	"github.com/redis/go-redis/v9"
)

func NewAccountManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *postgresql.AccountManagerSQL {
	return postgresql.NewAccountManagerSQL(db, redis, entityName)
}

func NewAccountFetchers(redis redis.UniversalClient, entityName string) *postgresql.AccountFetchers {
	return postgresql.NewAccountFetchers(redis, entityName)
}

func NewUpdateEmailManagerSQL(db *sql.DB, entityName string) *postgresql.UpdateEmailManagerSQL {
	return postgresql.NewUpdateEmailManagerSQL(db, entityName)
}

func NewResetPasswordSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *postgresql.ResetPasswordManagerSQL {
	return postgresql.NewResetPasswordManagerSQL(db, redis, entityName)
}

func NewJWTHandler(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan int) *lib.JWTHandler {
	return lib.NewJWTHandler(jwtSecret, jwtTokenIssuer, jwtTokenLifeSpan)
}

func NewAccountSQL() postgresql.AccountSQL {
	return postgresql.NewAccountSQL()
}
