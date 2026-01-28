package repository

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/types"
)

type UpdateEmailRepository struct {
	app               *config.App
	findByAccountStmt *sql.Stmt
}

func (em *UpdateEmailRepository) CreateRequest(ctx context.Context, db types.SQLExecutor, request *model.UpdateEmail) error {
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
	_, errInsert := db.ExecContext(ctx,
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

func (em *UpdateEmailRepository) UpdateRequest(ctx context.Context, db types.SQLExecutor, request *model.UpdateEmail) error {
	tableName := em.app.EntityName + "_update_email"
	query := `UPDATE ` + tableName + ` SET updated_at = $1, processed = $2 WHERE uuid = $3`
	_, errUpdate := db.ExecContext(ctx,
		query,
		request.GetUpdatedAt(),
		request.Processed,
		request.GetUUID(),
	)

	return errUpdate
}

func (em *UpdateEmailRepository) FindRequest(account *model.Account) (*model.UpdateEmail, error) {
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
			return nil, model.EmailChangeTokenNotFound
		}
		return nil, err
	}

	return updateEmailRequest, nil
}

func (em *UpdateEmailRepository) DeleteAllRequest(ctx context.Context, db types.SQLExecutor, account *model.Account) error {
	tableName := em.app.EntityName + "_update_email"
	query := `DELETE FROM ` + tableName + ` WHERE account_uuid = $1`
	_, errDelete := db.ExecContext(ctx, query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewUpdateEmailManager(readDB *sql.DB, app *config.App) *UpdateEmailRepository {
	tableName := app.EntityName + "_update_email"

	// always find the most recent ticket
	findByAccountStmt, errPrepare := readDB.Prepare(`SELECT * FROM ` + tableName + ` WHERE account_uuid = $1 ORDER BY created_at DESC LIMIT 1`)
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &UpdateEmailRepository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
