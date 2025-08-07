package lib

import (
	"database/sql"
	"errors"
	"github.com/21strive/redifu"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"time"
)

type AccountSQL struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	*Base
}

func (asql *AccountSQL) GenerateAccessToken(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan int) (string, error) {
	timeNow := time.Now().UTC()
	expirestAt := timeNow.Add(time.Hour * time.Duration(jwtTokenLifeSpan))

	userClaims := UserClaims{
		UUID:              asql.GetUUID(),
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

func (asql *AccountSQL) GenerateRefreshToken(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan int) (string, error) {
	timeNow := time.Now().UTC()
	expirestAt := timeNow.Add(time.Hour * time.Duration(jwtTokenLifeSpan))

	refreshTokenClaims := RefreshTokenClaims{
		UUID: asql.GetUUID(),
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func NewAccountSQL() *AccountSQL {
	account := &AccountSQL{
		Base: &Base{},
	}

	redifu.InitSQLItem(account)
	return account
}

type AccountManagerSQL struct {
	db         *sql.DB
	base       *redifu.Base[AccountSQL]
	entityName string
}

func (asql *AccountManagerSQL) SetEntityName(entityName string) {
	asql.entityName = entityName
}

func (asql *AccountManagerSQL) Create(account AccountSQL) error {
	query := "INSERT INTO $1 (name, username, password, email, avatar, suspended) VALUES ($2, $3, $4, $5, $6, $7)"
	_, errInsert := asql.db.Exec(query, asql.entityName, account.Name, account.Username, account.Password, account.Email, account.Avatar, account.Suspended)
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

func (asql *AccountManagerSQL) Update(account AccountSQL) error {
	query := "UPDATE $1 SET updatedat = $2, name = $3, username = $4, suspended = $5 WHERE id = $6"
	_, errUpdate := asql.db.Exec(query, asql.entityName, account.GetUpdatedAt(), account.Name, account.Username, account.Suspended)
	if errUpdate != nil {
		return errUpdate
	}
	return nil
}

func (asql *AccountManagerSQL) Delete(account AccountSQL) error {
	query := "DELETE FROM $1 WHERE uuid = $2"
	_, errDelete := asql.db.Exec(query, asql.entityName, account.UUID)
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func (asql *AccountManagerSQL) FindByUsername(username string) (*AccountSQL, error) {
	query := "SELECT uuid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE username = $1"
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
	return nil
}

func (asql *AccountManagerSQL) FindByRandId(randId string) (*AccountSQL, error) {
	query := "SELECT uuid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE randId = $1"
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

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByEmail(email string) (*AccountSQL, error) {
	query := "SELECT uuid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE email = $1"
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

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByUUID(uuid string) (*AccountSQL, error) {
	query := "SELECT uuid, createdat, updatedat, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE uuid = $1"
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

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func NewAccountManagerSQL(db *sql.DB, redis *redis.Client, entityName string) *AccountManagerSQL {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s")
	return &AccountManagerSQL{
		db:         db,
		base:       base,
		entityName: entityName,
	}
}

type AccountFetchers struct {
	base *redifu.Base[AccountSQL]
}

func (af *AccountFetchers) FetchByUsername(username string) (*AccountSQL, error) {
	account, err := af.base.Get(username)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchers) FetchByUUID(uuid string) (*AccountSQL, error) {
	account, err := af.base.Get(uuid)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchers) FetchByEmail(email string) (*AccountSQL, error) {
	account, err := af.base.Get(email)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchers) FetchByRandId(randId string) (*AccountSQL, error) {
	account, err := af.base.Get(randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
	}
	return &account, nil
}

func NewAccountFetchers(redis *redis.Client, entityName string) *AccountFetchers {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s")
	return &AccountFetchers{
		base: base,
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

	return account, nil
}
