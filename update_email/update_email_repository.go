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

func (em *Repository) CreateRequest(db shared.SQLExecutor, request *UpdateEmail) error {
	tableName := em.app.EntityName + "_update_email"
	query := `INSERT INTO ` + tableName + ` 
		(
			uuid, 
			randId, 
			created_at, 
			updated_at, 
			account_uuid, 
			previous_email_address, 
			new_email_address, 
			reset_token,
			revoke_token,
			processed,
			expired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, errInsert := db.Exec(
		query,
		request.GetUUID(),
		request.GetRandId(),
		request.GetCreatedAt(),
		request.GetUpdatedAt(),
		request.AccountUUID,
		request.PreviousEmailAddress,
		request.NewEmailAddress,
		request.Token,
		request.RevokeToken,
		request.Processed,
		request.ExpiredAt)

	return errInsert
}

func (em *Repository) UpdateRequest(db shared.SQLExecutor, request *UpdateEmail) error {
	tableName := em.app.EntityName + "_update_email"
	query := `UPDATE ` + tableName + ` SET updated_at = $1, processed = $2 WHERE uuid = $3`
	_, errUpdate := db.Exec(
		query,
		request.GetUpdatedAt(),
		request.Processed,
		request.GetUUID(),
	)

	return errUpdate
}

func (em *Repository) FindRequest(account *account.Account) (*UpdateEmail, error) {
	row := em.findByAccountStmt.QueryRow(account.GetUUID())
	updateEmailRequest := New()
	err := row.Scan(
		&updateEmailRequest.UUID,
		&updateEmailRequest.RandId,
		&updateEmailRequest.CreatedAt,
		&updateEmailRequest.UpdatedAt,
		&updateEmailRequest.AccountUUID,
		&updateEmailRequest.PreviousEmailAddress,
		&updateEmailRequest.NewEmailAddress,
		&updateEmailRequest.Token,
		&updateEmailRequest.RevokeToken,
		&updateEmailRequest.Processed,
		&updateEmailRequest.ExpiredAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, TicketNotFound
		}
		return nil, err
	}

	return updateEmailRequest, nil
}

func (em *Repository) DeleteAll(db shared.SQLExecutor, account *account.Account) error {
	tableName := em.app.EntityName + "_update_email"
	query := `DELETE FROM ` + tableName + ` WHERE account_uuid = $1`
	_, errDelete := db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewUpdateEmailManager(readDB *sql.DB, app *config.App) *Repository {
	tableName := app.EntityName + "_update_email"

	// always find the most recent ticket
	findByAccountStmt, errPrepare := readDB.Prepare(`SELECT * FROM ` + tableName + ` WHERE account_uuid = $1 ORDER BY created_at DESC LIMIT 1`)
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &Repository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
