package update_email

import (
	"database/sql"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/shared"
	"time"
)

type UpdateEmailManagerSQL struct {
	db         *sql.DB
	entityName string
}

func (em *UpdateEmailManagerSQL) CreateRequest(account account.Account, newEmailAddress string) (*UpdateEmail, error) {
	updateEmailRequest := NewUpdateEmailRequestSQL()
	updateEmailRequest.SetPreviousEmailAddress(account.Base.Email)
	updateEmailRequest.SetNewEmailAddress(newEmailAddress)
	updateEmailRequest.SetResetToken()
	updateEmailRequest.SetExpiration()

	tableName := em.entityName + "_update_email"

	query := `INSERT INTO ` + tableName + ` (uuid, randId, created_at, updated_at, account_uuid, previous_email_address, new_email_address, update_token, expired_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, errInsert := em.db.Exec(
		query,
		updateEmailRequest.GetUUID(),
		updateEmailRequest.GetRandId(),
		updateEmailRequest.GetCreatedAt(),
		updateEmailRequest.GetUpdatedAt(),
		updateEmailRequest.AccountUUID,
		updateEmailRequest.PreviousEmailAddress,
		updateEmailRequest.NewEmailAddress,
		updateEmailRequest.UpdateToken,
		updateEmailRequest.ExpiredAt)
	if errInsert != nil {
		return nil, errInsert
	}

	return &updateEmailRequest, nil
}

func (em *UpdateEmailManagerSQL) FindRequest(account account.Account) (*UpdateEmail, error) {
	tableName := em.entityName + "_update_email"
	query := `SELECT * FROM ` + tableName + ` WHERE account_uuid = $1`
	row := em.db.QueryRow(query, account.GetUUID())
	updateEmailRequest := NewUpdateEmailRequestSQL()
	err := row.Scan(
		&updateEmailRequest.SQLItem.UUID,
		&updateEmailRequest.SQLItem.RandId,
		&updateEmailRequest.SQLItem.CreatedAt,
		&updateEmailRequest.SQLItem.UpdatedAt,
		&updateEmailRequest.AccountUUID,
		&updateEmailRequest.PreviousEmailAddress,
		&updateEmailRequest.NewEmailAddress,
		&updateEmailRequest.UpdateToken,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if updateEmailRequest.ExpiredAt.Before(time.Now().UTC()) {
		em.DeleteRequest(&updateEmailRequest)
		newUpdateEmailRequest, err := em.CreateRequest(account, updateEmailRequest.NewEmailAddress)
		if err != nil {
			return nil, err
		}
		return newUpdateEmailRequest, nil
	}

	return &updateEmailRequest, nil
}

func (em *UpdateEmailManagerSQL) DeleteRequest(request *UpdateEmail) error {
	tableName := em.entityName + "_update_email"
	query := `DELETE FROM ` + tableName + ` WHERE uuid = $1`
	_, errDelete := em.db.Exec(query, request.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func (em *UpdateEmailManagerSQL) ValidateRequest(account account.Account, updateToken string) error {
	request, errFind := em.FindRequest(account)
	if errFind != nil {
		return errFind
	}
	if request == nil {
		return shared.RequestNotFound
	}
	errValidate := request.Validate(updateToken)
	if errValidate != nil {
		if errValidate == shared.RequestExpired {
			em.DeleteRequest(request)
			return shared.RequestExpired
		}
		return errValidate
	}
	return nil
}

func NewUpdateEmailManagerSQL(db *sql.DB, entityName string) *UpdateEmailManagerSQL {
	return &UpdateEmailManagerSQL{
		db:         db,
		entityName: entityName,
	}
}
