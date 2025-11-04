package provider

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/shared"
)

type Repository struct {
	findBySubStmt *sql.Stmt
	app           *config.App
}

func (r *Repository) Create(db shared.SQLExecutor, provider *Provider) error {
	tableName := r.app.EntityName + "_provider"
	query := "INSERT INTO " + tableName + " (uuid, randid, created_at, updated_at, name, email, provider_uuid, sub, provider) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, errExec := db.Exec(
		query,
		provider.GetUUID(),
		provider.GetRandId(),
		provider.GetCreatedAt(),
		provider.GetUpdatedAt(),
		provider.Name,
		provider.Email,
		provider.Uuid,
		provider.Sub,
		provider.Provider,
	)
	if errExec != nil {
		return errExec
	}

	return nil
}

func (r *Repository) scanProvider(scanner interface {
	Scan(dest ...interface{}) error
}) (*Provider, error) {
	provider := NewProvider()
	err := scanner.Scan(
		&provider.UUID,
		&provider.RandId,
		&provider.CreatedAt,
		&provider.UpdatedAt,
		&provider.Name,
		&provider.Email,
		&provider.Uuid,
		&provider.Sub,
		&provider.Provider,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ProviderNotFound
		}
		return nil, err
	}

	return provider, nil
}

func (r *Repository) Find(sub string) (*Provider, error) {
	row := r.findBySubStmt.QueryRow(sub)
	return r.scanProvider(row)
}

func (r *Repository) Delete(db shared.SQLExecutor, provider *Provider) error {
	tableName := r.app.EntityName + "_provider"
	query := "DELETE FROM " + tableName + " WHERE uuid = $1"
	_, errExec := db.Exec(query, provider.GetUUID())
	if errExec != nil {
		return errExec
	}

	return nil
}

func NewRepository(readDB *sql.DB, app *config.App) *Repository {
	tableName := app.EntityName + "_provider"
	findBySubStmt, errPrepare := readDB.Prepare("SELECT uuid, randid, created_at, updated_at, name, email, provider_uuid, sub, provider FROM " + tableName + " WHERE sub = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &Repository{
		findBySubStmt: findBySubStmt,
		app:           app,
	}
}
