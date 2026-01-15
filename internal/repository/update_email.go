package repository

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
)

var UpdateEmailTicketNotFound = errors.New("Update email ticket not found")
var InvalidUpdateEmailToken = errors.New("Invalid token")

type EmailRepository struct {
	app               *config.App
	findByAccountStmt *sql.Stmt
}

func (em *EmailRepository) CreateRequest(db database.SQLExecutor, request *model.UpdateEmail) error {
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

func (em *EmailRepository) UpdateRequest(db database.SQLExecutor, request *model.UpdateEmail) error {
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

func (em *EmailRepository) FindRequest(account *model.Account) (*model.UpdateEmail, error) {
	row := em.findByAccountStmt.QueryRow(account.GetUUID())
	updateEmailRequest := model.NewUpdateEmailRequest()
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
			return nil, UpdateEmailTicketNotFound
		}
		return nil, err
	}

	return updateEmailRequest, nil
}

func (em *EmailRepository) DeleteAllRequest(db database.SQLExecutor, account *model.Account) error {
	tableName := em.app.EntityName + "_update_email"
	query := `DELETE FROM ` + tableName + ` WHERE account_uuid = $1`
	_, errDelete := db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewUpdateEmailManager(readDB *sql.DB, app *config.App) *EmailRepository {
	tableName := app.EntityName + "_update_email"

	// always find the most recent ticket
	findByAccountStmt, errPrepare := readDB.Prepare(`SELECT * FROM ` + tableName + ` WHERE account_uuid = $1 ORDER BY created_at DESC LIMIT 1`)
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &EmailRepository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
