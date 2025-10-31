package reset_password

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

func (ar *Repository) Create(db shared.SQLExecutor, request *ResetPassword) error {
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

func (ar *Repository) Find(account *account.Account) (*ResetPassword, error) {
	row := ar.findByAccountStmt.QueryRow(account.GetUUID())
	resetPasswordRequest := New()
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
			return nil, TicketNotFound
		}
		return nil, err
	}

	return resetPasswordRequest, nil
}

func (ar *Repository) Delete(db shared.SQLExecutor, request *ResetPassword) error {
	tableName := ar.app.EntityName + "_reset_password"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errDelete := db.Exec(query, request.GetUUID())
	if errDelete != nil {
		return errDelete
	}
	return nil
}

func NewRepository(readDB *sql.DB, app *config.App) *Repository {
	tableName := app.EntityName + "_reset_password"

	// always find the most recent ticket
	findByAccountStmt, errPrepare := readDB.Prepare("SELECT * FROM " + tableName + " WHERE account_uuid = $1 ORDER BY created_at DESC LIMIT 1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &Repository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
