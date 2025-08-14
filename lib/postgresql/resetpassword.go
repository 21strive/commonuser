package postgresql

import (
	"database/sql"
	"github.com/21strive/commonuser/definition"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type ResetPasswordRequestSQL struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	AccountUUID     string    `db:"accountuuid"`
	Token           string    `db:"token"`
	ExpiredAt       time.Time `db:"expiredat"`
}

func (rpsql *ResetPasswordRequestSQL) SetAccountUUID(account *AccountSQL) {
	rpsql.AccountUUID = account.GetUUID()
}

func (rpsql *ResetPasswordRequestSQL) SetToken() {
	rpsql.Token = redifu.RandId()
}

func (rpsql *ResetPasswordRequestSQL) SetExpiredAt() {
	rpsql.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (rpsql *ResetPasswordRequestSQL) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(rpsql.ExpiredAt) {
		return definition.RequestExpired
	}
	if rpsql.Token != token {
		return definition.InvalidToken
	}
	return nil
}

func NewResetPasswordSQL() *ResetPasswordRequestSQL {
	request := &ResetPasswordRequestSQL{}
	redifu.InitSQLItem(request)
	return request
}

type ResetPasswordManagerSQL struct {
	base       *redifu.Base[AccountSQL]
	db         *sql.DB
	entityName string
}

func (ar *ResetPasswordManagerSQL) Create(account *AccountSQL) (*ResetPasswordRequestSQL, error) {
	requestResetPassword := NewResetPasswordSQL()
	requestResetPassword.SetAccountUUID(account)
	requestResetPassword.SetToken()
	requestResetPassword.SetExpiredAt()

	tableName := ar.entityName + "ResetPassword"
	query := `INSERT INTO $1 (uuid, randId, createdat, updatedat, accountuuid, token, expiredat) VALUES ($2, $3, $4, $5, $6, $7, $8)`
	_, errInsert := ar.db.Exec(
		query,
		tableName,
		requestResetPassword.GetUUID(),
		requestResetPassword.GetRandId(),
		requestResetPassword.GetCreatedAt(),
		requestResetPassword.GetUpdatedAt(),
		requestResetPassword.AccountUUID,
		requestResetPassword.Token,
		requestResetPassword.ExpiredAt)

	if errInsert != nil {
		return nil, errInsert
	}

	return requestResetPassword, nil
}

func (ar *ResetPasswordManagerSQL) Find(account *AccountSQL) (*ResetPasswordRequestSQL, error) {
	tableName := ar.entityName + "ResetPassword"
	query := "SELECT * FROM " + tableName + "ResetPassword WHERE accountuuid = $1"
	row := ar.db.QueryRow(query, account.Email)
	resetPasswordRequest := NewResetPasswordSQL()
	err := row.Scan(
		&resetPasswordRequest.SQLItem.UUID,
		&resetPasswordRequest.SQLItem.RandId,
		&resetPasswordRequest.SQLItem.CreatedAt,
		&resetPasswordRequest.SQLItem.UpdatedAt,
		&resetPasswordRequest.AccountUUID,
		&resetPasswordRequest.Token,
		&resetPasswordRequest.ExpiredAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if resetPasswordRequest.ExpiredAt.Before(time.Now().UTC()) {
		ar.Delete(resetPasswordRequest)
		newResetPasswordRequest, err := ar.Create(account)
		if err != nil {
			return nil, err
		}
		return newResetPasswordRequest, nil
	} else {
		return nil, definition.RequestExist
	}

	return resetPasswordRequest, nil
}

func (ar *ResetPasswordManagerSQL) Delete(requestSQL *ResetPasswordRequestSQL) error {
	tableName := ar.entityName + "ResetPassword"
	query := "DELETE FROM " + tableName + "ResetPassword WHERE uuid = $1"
	_, errDelete := ar.db.Exec(query, requestSQL.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewResetPasswordManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *ResetPasswordManagerSQL {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s")
	return &ResetPasswordManagerSQL{
		base: base,
		db:   db,
	}
}
