package verification

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

func (r *Repository) Close() {
	r.findByAccountStmt.Close()
}

func (r *Repository) Create(db shared.SQLExecutor, verification *Verification) error {
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

func (r *Repository) Update(db shared.SQLExecutor, verification *Verification) error {
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

func (r *Repository) Delete(db shared.SQLExecutor, verification *Verification) error {
	tableName := r.app.EntityName + "_verification"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errExec := db.Exec(query, verification.GetUUID())
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *Repository) FindByAccount(account *account.Account) (*Verification, error) {
	return VerificationRowScanner(r.findByAccountStmt.QueryRow(account.GetUUID()))
}

func VerificationRowScanner(row *sql.Row) (*Verification, error) {
	verification := New()
	err := row.Scan(
		&verification.UUID,
		&verification.CreatedAt,
		&verification.UpdatedAt,
		&verification.AccountUUID,
		&verification.Code,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, VerificationNotFound
		}

		return nil, err
	}

	return verification, nil
}

func NewRepository(readDB *sql.DB, app *config.App) *Repository {
	tableName := app.EntityName + "_verification"
	findByAccountStmt, errPrepare := readDB.Prepare("SELECT * FROM " + tableName + " WHERE account_uuid = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &Repository{
		findByAccountStmt: findByAccountStmt,
	}
}
