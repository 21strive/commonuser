package repository

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
)

type ResetPasswordRepository struct {
	app               *config.App
	findByAccountStmt *sql.Stmt
}

func (ar *ResetPasswordRepository) CreateRequest(db database.SQLExecutor, request *model.ResetPassword) error {
	tableName := ar.app.EntityName + "_reset_password"
	query := `INSERT INTO ` + tableName + ` (
		uuid, 
		randid, 
		created_at, 
		updated_at, 
		account_uuid, 
		token, 
		expired_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, errInsert := db.Exec(
		query,
		request.GetUUID(),
		request.GetRandId(),
		request.GetCreatedAt(),
		request.GetUpdatedAt(),
		request.AccountUUID,
		request.Token,
		request.ExpiredAt)

	return errInsert
}

func (ar *ResetPasswordRepository) FindRequest(account *model.Account) (*model.ResetPassword, error) {
	row := ar.findByAccountStmt.QueryRow(account.GetUUID())
	resetPasswordRequest := model.NewResetPasswordRequest()
	err := row.Scan(
		&resetPasswordRequest.UUID,
		&resetPasswordRequest.RandId,
		&resetPasswordRequest.CreatedAt,
		&resetPasswordRequest.UpdatedAt,
		&resetPasswordRequest.AccountUUID,
		&resetPasswordRequest.Token,
		&resetPasswordRequest.ExpiredAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ResetPasswordTicketNotFound
		}
		return nil, err
	}

	return resetPasswordRequest, nil
}

func (ar *ResetPasswordRepository) DeleteAllRequests(db database.SQLExecutor, account *model.Account) error {
	tableName := ar.app.EntityName + "_reset_password"
	query := "DELETE FROM " + tableName + " WHERE account_uuid = $1"
	_, errDelete := db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewResetPasswordRepository(readDB *sql.DB, app *config.App) *ResetPasswordRepository {
	tableName := app.EntityName + "_reset_password"

	// always find the most recent ticket
	findByAccountStmt, errPrepare := readDB.Prepare("SELECT * FROM " + tableName + " WHERE account_uuid = $1 ORDER BY created_at DESC LIMIT 1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &ResetPasswordRepository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
