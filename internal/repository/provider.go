package repository

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
)

type ProviderRepository struct {
	findBySubStmt *sql.Stmt
	app           *config.App
}

func (r *ProviderRepository) Create(db database.SQLExecutor, provider *model.Provider) error {
	tableName := r.app.EntityName + "_provider"
	query := "INSERT INTO " + tableName + " (uuid, randid, created_at, updated_at, name, email, sub, issuer, account_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, errExec := db.Exec(
		query,
		provider.GetUUID(),
		provider.GetRandId(),
		provider.GetCreatedAt(),
		provider.GetUpdatedAt(),
		provider.Name,
		provider.Email,
		provider.Sub,
		provider.Issuer,
		provider.AccountUUID,
	)
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *ProviderRepository) scanProvider(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Provider, error) {
	provider := model.NewProvider()
	err := scanner.Scan(
		&provider.UUID,
		&provider.RandId,
		&provider.CreatedAt,
		&provider.UpdatedAt,
		&provider.Name,
		&provider.Email,
		&provider.Sub,
		&provider.Issuer,
		&provider.AccountUUID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ProviderNotFound
		}
		return nil, err
	}

	return provider, nil
}

func (r *ProviderRepository) Find(sub string, issuer string) (*model.Provider, error) {
	row := r.findBySubStmt.QueryRow(sub, issuer)
	return r.scanProvider(row)
}

func (r *ProviderRepository) Delete(db database.SQLExecutor, provider *model.Provider) error {
	tableName := r.app.EntityName + "_provider"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errExec := db.Exec(query, provider.GetUUID())
	if errExec != nil {
		return errExec
	}

	return nil
}

func NewProviderRepository(readDB *sql.DB, app *config.App) *ProviderRepository {
	tableName := app.EntityName + "_provider"
	findBySubStmt, errPrepare := readDB.Prepare("SELECT uuid, randid, created_at, updated_at, name, email, sub, issuer, account_uuid FROM " + tableName + " WHERE sub = $1 AND issuer = $2")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &ProviderRepository{
		findBySubStmt: findBySubStmt,
		app:           app,
	}
}
