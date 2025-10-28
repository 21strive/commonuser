package update_email

import (
	"database/sql"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/shared"
)

type Repository struct {
	app               *config.App
	findByAccountStmt *sql.Stmt
}

func (em *Repository) CreateRequest(db shared.SQLExecutor, account *account.Account, newEmailAddress string) (*UpdateEmail, error) {
	updateEmailRequest := NewUpdateEmailRequestSQL()
	updateEmailRequest.SetPreviousEmailAddress(account.Base.Email)
	updateEmailRequest.SetNewEmailAddress(newEmailAddress)
	updateEmailRequest.SetToken()
	updateEmailRequest.SetExpiration()

	tableName := em.app.EntityName + "_update_email"

	query := `INSERT INTO ` + tableName + ` (uuid, randId, created_at, updated_at, account_uuid, previous_email_address, new_email_address, update_token, expired_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, errInsert := db.Exec(
		query,
		updateEmailRequest.GetUUID(),
		updateEmailRequest.GetRandId(),
		updateEmailRequest.GetCreatedAt(),
		updateEmailRequest.GetUpdatedAt(),
		updateEmailRequest.AccountUUID,
		updateEmailRequest.PreviousEmailAddress,
		updateEmailRequest.NewEmailAddress,
		updateEmailRequest.Token,
		updateEmailRequest.ExpiredAt)
	if errInsert != nil {
		return nil, errInsert
	}

	return &updateEmailRequest, nil
}

func (em *Repository) FindRequest(account *account.Account) (*UpdateEmail, error) {
	row := em.findByAccountStmt.QueryRow(account.GetUUID())
	updateEmailRequest := NewUpdateEmailRequestSQL()
	err := row.Scan(
		&updateEmailRequest.UUID,
		&updateEmailRequest.RandId,
		&updateEmailRequest.CreatedAt,
		&updateEmailRequest.UpdatedAt,
		&updateEmailRequest.AccountUUID,
		&updateEmailRequest.PreviousEmailAddress,
		&updateEmailRequest.NewEmailAddress,
		&updateEmailRequest.Token,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, TicketNotFound
		}
		return nil, err
	}

	return &updateEmailRequest, nil
}

func (em *Repository) DeleteRequest(db shared.SQLExecutor, request *UpdateEmail) error {
	tableName := em.app.EntityName + "_update_email"
	query := `DELETE FROM ` + tableName + ` WHERE uuid = $1`
	_, errDelete := db.Exec(query, request.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewUpdateEmailManagerSQL(readDB *sql.DB, app *config.App) *Repository {
	tableName := app.EntityName + "_update_email"
	findByAccountStmt, errPrepare := readDB.Prepare(`SELECT * FROM ` + tableName + ` WHERE account_uuid = $1`)
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &Repository{
		findByAccountStmt: findByAccountStmt,
	}
}
