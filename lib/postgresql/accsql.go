package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/lib"
	"github.com/21strive/redifu"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"time"
)

type AccountSQL struct {
	*redifu.SQLItem
	lib.Base
}

func (asql *AccountSQL) GenerateAccessToken(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan time.Duration) (string, error) {
	timeNow := time.Now().UTC()
	expirestAt := timeNow.Add(jwtTokenLifeSpan)

	userClaims := lib.UserClaims{
		UUID:              asql.GetUUID(),
		RandId:            asql.GetRandId(),
		Name:              asql.Name,
		Username:          asql.Username,
		Email:             asql.Email,
		Avatar:            asql.Avatar,
		PasswordUpdatedAt: asql.PasswordUpdatedAt,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: jwtTokenIssuer,
			IssuedAt: &jwt.NumericDate{
				Time: timeNow,
			},
			ExpiresAt: &jwt.NumericDate{
				Time: expirestAt,
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func NewAccountSQL() AccountSQL {
	account := AccountSQL{
		Base: lib.Base{},
	}
	redifu.InitSQLItem(&account)
	return account
}

type AccountManagerSQL struct {
	db         *sql.DB
	base       *redifu.Base[AccountSQL]
	redis      redis.UniversalClient
	entityName string
}

func (asql *AccountManagerSQL) Create(account AccountSQL) error {
	query := "INSERT INTO " + asql.entityName + " (uuid, randid, createdat, updatedat, name, username, password, email, avatar, suspended) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	_, errInsert := asql.db.Exec(query, account.GetUUID(), account.GetRandId(), account.GetCreatedAt(), account.GetUpdatedAt(), account.Name, account.Username, account.Password, account.Email, account.Avatar, account.Suspended)
	if errInsert != nil {
		return errInsert
	}
	if account.Username != "" {
		asql.base.Set(account, account.Username)
	} else {
		asql.base.Set(account)
	}
	return nil
}

func (asql *AccountManagerSQL) Patch(account AccountSQL) error {
	query := "UPDATE " + asql.entityName + " SET updatedat = $1, name = $2, avatar = $3 WHERE uuid = $4"
	_, errUpdate := asql.db.Exec(query, account.GetUpdatedAt(), account.Name, account.Avatar, account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}
	return nil
}

func (asql *AccountManagerSQL) Delete(account AccountSQL) error {
	query := "DELETE FROM " + asql.entityName + " WHERE uuid = $1"
	_, errDelete := asql.db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func (asql *AccountManagerSQL) setAccountToCache(account AccountSQL) error {
	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	if account.Username != "" {
		key := asql.entityName + ":username:" + account.Username
		setReference := asql.redis.Set(context.TODO(), key, account.GetRandId(), 7*24*time.Hour)
		if setReference.Err() != nil {
			return setReference.Err()
		}
	}
	return nil
}

func (asql *AccountManagerSQL) FindByUsername(username string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE username = $1"
	return findOneAccount(asql.db, query, username)
}

func (asql *AccountManagerSQL) SeedByUsername(username string) error {
	account, err := asql.FindByUsername(username)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccountToCache(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByRandId(randId string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE randId = $1"
	return findOneAccount(asql.db, query, randId)
}

func (asql *AccountManagerSQL) SeedByRandId(randId string) error {
	account, err := asql.FindByRandId(randId)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccountToCache(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByEmail(email string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE email = $1"
	return findOneAccount(asql.db, query, email)
}

func (asql *AccountManagerSQL) SeedByEmail(email string) error {
	account, err := asql.FindByEmail(email)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccountToCache(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByUUID(uuid string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE uuid = $1"
	return findOneAccount(asql.db, query, uuid)
}

func (asql *AccountManagerSQL) SeedByUUID(uuid string) error {
	account, err := asql.FindByUUID(uuid)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccountToCache(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func NewAccountManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *AccountManagerSQL {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s")
	return &AccountManagerSQL{
		db:         db,
		base:       base,
		entityName: entityName,
	}
}

func findOneAccount(db *sql.DB, query string, param string) (*AccountSQL, error) {
	row := db.QueryRow(query, param)
	account := NewAccountSQL()
	err := row.Scan(
		&account.SQLItem.UUID,
		&account.SQLItem.RandId,
		&account.SQLItem.CreatedAt,
		&account.SQLItem.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
		&account.Base.Suspended,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}
