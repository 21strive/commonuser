package postgresql

import (
	"database/sql"
	"github.com/21strive/commonuser/definition"
	"github.com/21strive/redifu"
	"time"
)

type UpdateEmailRequestSQL struct {
	*redifu.SQLItem      `bson:",inline" json:",inline"`
	AccountUUID          string    `db:"account_uuid"`
	PreviousEmailAddress string    `db:"previous_email_address"`
	NewEmailAddress      string    `db:"new_email_address"`
	UpdateToken          string    `db:"update_token"`
	ExpiredAt            time.Time `db:"expired_at"`
}

func (ue *UpdateEmailRequestSQL) SetAccountUUID(account *AccountSQL) {
	ue.AccountUUID = account.UUID
}

func (ue *UpdateEmailRequestSQL) SetPreviousEmailAddress(email string) {
	ue.PreviousEmailAddress = email
}

func (ue *UpdateEmailRequestSQL) SetNewEmailAddress(email string) {
	ue.NewEmailAddress = email
}

func (ue *UpdateEmailRequestSQL) SetResetToken() {
	token := redifu.RandId()
	ue.UpdateToken = token
}

func (ue *UpdateEmailRequestSQL) SetExpiration() {
	ue.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (ue *UpdateEmailRequestSQL) Validate(updateToken string) error {
	time := time.Now().UTC()
	if time.After(ue.ExpiredAt) {
		return definition.RequestExpired
	}

	if ue.UpdateToken != updateToken {
		return definition.InvalidToken
	}
	return nil
}

func NewUpdateEmailRequestSQL() *UpdateEmailRequestSQL {
	ue := &UpdateEmailRequestSQL{}
	redifu.InitSQLItem(ue)
	return ue
}

type UpdateEmailManagerSQL struct {
	db         *sql.DB
	entityName string
}

func (em *UpdateEmailManagerSQL) CreateRequest(account AccountSQL, newEmailAddress string) (*UpdateEmailRequestSQL, error) {
	updateEmailRequest := NewUpdateEmailRequestSQL()
	updateEmailRequest.SetPreviousEmailAddress(account.Base.Email)
	updateEmailRequest.SetNewEmailAddress(newEmailAddress)
	updateEmailRequest.SetResetToken()
	updateEmailRequest.SetExpiration()

	tableName := em.entityName + "UpdateEmailManagerSQL"

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

	return updateEmailRequest, nil
}

func (em *UpdateEmailManagerSQL) FindRequest(account AccountSQL) (*UpdateEmailRequestSQL, error) {
	query := `SELECT * FROM ` + em.entityName + `update_email WHERE account_uuid = $1`
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
		em.DeleteRequest(updateEmailRequest)
		newUpdateEmailRequest, err := em.CreateRequest(account, updateEmailRequest.NewEmailAddress)
		if err != nil {
			return nil, err
		}
		return newUpdateEmailRequest, nil
	} else {
		return nil, definition.RequestExist
	}
	return updateEmailRequest, nil
}

func (em *UpdateEmailManagerSQL) DeleteRequest(request *UpdateEmailRequestSQL) error {
	query := `DELETE FROM ` + em.entityName + `UpdateEmailManagerSQL WHERE uuid = $1`
	_, errDelete := em.db.Exec(query, request.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func (em *UpdateEmailManagerSQL) ValidateRequest(account AccountSQL, updateToken string) error {
	request, errFind := em.FindRequest(account)
	if errFind != nil {
		return errFind
	}
	if request == nil {
		return definition.RequestNotFound
	}
	errValidate := request.Validate(updateToken)
	if errValidate != nil {
		if errValidate == definition.RequestExpired {
			em.DeleteRequest(request)
			return definition.RequestExpired
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
