package commonuser

import (
	"database/sql"
	"github.com/21strive/commonuser/lib"
	"github.com/redis/go-redis/v9"
)

func NewAccountManagerSQL(db *sql.DB, redis *redis.Client, entityName string) *lib.AccountManagerSQL {
	return lib.NewAccountManagerSQL(db, redis, entityName)
}

func NewUpdateEmailManagerSQL(db *sql.DB, entityName string) *lib.UpdateEmailManagerSQL {
	return lib.NewUpdateEmailManagerSQL(db, entityName)
}

func NewResetPasswordSQL(db *sql.DB, redis *redis.Client, entityName string) *lib.ResetPasswordManagerSQL {
	return lib.NewResetPasswordManagerSQL(db, redis, entityName)
}

func NewJWTHandler(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan int) *lib.JWTHandler {
	return lib.NewJWTHandler(jwtSecret, jwtTokenIssuer, jwtTokenLifeSpan)
}
