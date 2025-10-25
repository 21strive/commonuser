package reset_password

import (
	"database/sql"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type ResetPasswordManagerSQL struct {
	base       *redifu.Base[account.Account]
	db         *sql.DB
	entityName string
}

func (ar *ResetPasswordManagerSQL) Create(account *account.Account) (*ResetPassword, error) {
	requestResetPassword := NewResetPasswordSQL()
	requestResetPassword.SetAccountUUID(account)
	requestResetPassword.SetToken()
	requestResetPassword.SetExpiredAt()

	tableName := ar.entityName + "_reset_password"
	query := `INSERT INTO $1 (uuid, randId, created_at, updated_at, account_uuid, token, expiredat) VALUES ($2, $3, $4, $5, $6, $7, $8)`
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

	return &requestResetPassword, nil
}

func (ar *ResetPasswordManagerSQL) Find(account *account.Account) (*ResetPassword, error) {
	tableName := ar.entityName + "_reset_password"
	query := "SELECT * FROM " + tableName + " WHERE accountuuid = $1"
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
		return nil, shared.RequestNotFound
	}

	if resetPasswordRequest.ExpiredAt.Before(time.Now().UTC()) {
		ar.Delete(&resetPasswordRequest)
		newResetPasswordRequest, err := ar.Create(account)
		if err != nil {
			return nil, err
		}
		return newResetPasswordRequest, nil
	}

	return &resetPasswordRequest, nil
}

func (ar *ResetPasswordManagerSQL) Delete(requestSQL *ResetPassword) error {
	tableName := ar.entityName + "_reset_password"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errDelete := ar.db.Exec(query, requestSQL.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewResetPasswordManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *ResetPasswordManagerSQL {
	base := redifu.NewBase[account.Account](redis, entityName+":%s", shared.BaseTTL)
	return &ResetPasswordManagerSQL{
		base: base,
		db:   db,
	}
}
