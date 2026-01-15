package repository

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
)

type VerificationRepository struct {
	app               *config.App
	findByAccountStmt *sql.Stmt
}

func (r *VerificationRepository) Close() {
	r.findByAccountStmt.Close()
}

func (r *VerificationRepository) Create(db database.SQLExecutor, verification *model.Verification) error {
	tableName := r.app.EntityName + "_verification"
	query := "INSERT INTO " + tableName + " VALUES ($1, $2, $3, $4, $5, $6)"
	_, errExec := db.Exec(
		query,
		verification.GetUUID(),
		verification.GetRandId(),
		verification.GetCreatedAt(),
		verification.GetUpdatedAt(),
		verification.AccountUUID,
		verification.Code)
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *VerificationRepository) Update(db database.SQLExecutor, verification *model.Verification) error {
	tableName := r.app.EntityName + "_verification"
	query := "UPDATE " + tableName + " SET verification_hash = $1 WHERE uuid = $2"
	_, errExec := db.Exec(
		query,
		verification.Code,
		verification.GetUUID(),
	)
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *VerificationRepository) Delete(db database.SQLExecutor, verification *model.Verification) error {
	tableName := r.app.EntityName + "_verification"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errExec := db.Exec(query, verification.GetUUID())
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *VerificationRepository) FindByAccount(account *model.Account) (*model.Verification, error) {
	return VerificationRowScanner(r.findByAccountStmt.QueryRow(account.GetUUID()))
}

func VerificationRowScanner(row *sql.Row) (*model.Verification, error) {
	verification := model.NewVerification()
	err := row.Scan(
		&verification.UUID,
		&verification.RandId,
		&verification.CreatedAt,
		&verification.UpdatedAt,
		&verification.AccountUUID,
		&verification.Code,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.VerificationNotFound
		}

		return nil, err
	}

	return verification, nil
}

func NewVerificationRepository(readDB *sql.DB, app *config.App) *VerificationRepository {
	tableName := app.EntityName + "_verification"
	findByAccountStmt, errPrepare := readDB.Prepare("SELECT * FROM " + tableName + " WHERE account_uuid = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &VerificationRepository{
		findByAccountStmt: findByAccountStmt,
		app:               app,
	}
}
