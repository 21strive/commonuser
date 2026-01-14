package repository

import (
	"database/sql"
	"errors"
	account2 "github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

// key := asql.entityName + ":username:" + account.Username

type AccountRepository struct {
	redis               redis.UniversalClient
	base                *redifu.Base[*model.Account]
	baseReference       *redifu.Base[*model.AccountReference]
	sortedAccount       *redifu.Sorted[*model.Account]
	sortedAccountSeeder *redifu.SortedSeeder[*model.Account]
	findByUsernameStmt  *sql.Stmt
	findByRandIdStmt    *sql.Stmt
	findByEmailStmt     *sql.Stmt
	findByUUIDStmt      *sql.Stmt
	app                 *config.App
}

func (asql *AccountRepository) GetBase() *redifu.Base[*model.Account] {
	return asql.base
}

func (asql *AccountRepository) Close() {
	asql.findByUsernameStmt.Close()
	asql.findByRandIdStmt.Close()
	asql.findByEmailStmt.Close()
	asql.findByUUIDStmt.Close()
}

func (asql *AccountRepository) Create(db database.SQLExecutor, account *model.Account) error {
	query := "INSERT INTO " + asql.app.EntityName + ` (
		uuid, 
		randid, 
		created_at, 
		updated_at, 
		name, 
		username, 
		password, 
		email, 	
		avatar
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, errInsert := db.Exec(
		query,
		account.GetUUID(),
		account.GetRandId(),
		account.GetCreatedAt(),
		account.GetUpdatedAt(),
		account.Name,
		account.Username,
		account.Password,
		account.Email,
		account.Avatar,
	)

	if errInsert != nil {
		return errInsert
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	asql.baseReference.DelBlank(account.Username)
	asql.base.DelBlank(account.GetRandId())
	asql.sortedAccount.AddItem(account, nil)
	return nil
}

func (asql *AccountRepository) Update(db database.SQLExecutor, account *model.Account) error {
	query := "UPDATE " + asql.app.EntityName +
		" SET updated_at = $1, name = $2, username = $3, password = $4, email = $5, avatar = $6, email_verified = $7 WHERE uuid = $8"
	_, errUpdate := db.Exec(
		query,
		account.GetUpdatedAt(),
		account.Name,
		account.Username,
		account.Password,
		account.Email,
		account.Avatar,
		account.EmailVerified,
		account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	return nil
}

func (asql *AccountRepository) UpdateReference(account *model.Account, oldUsername string, newUsername string) error {
	ref, errGet := asql.baseReference.Get(oldUsername)
	if errGet != nil {
		return errGet
	}

	err := asql.baseReference.Del(ref)
	if err != nil {
		return err
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSet := asql.baseReference.Set(accountReference, newUsername)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (asql *AccountRepository) Delete(db database.SQLExecutor, account *model.Account) error {
	query := "DELETE FROM " + asql.app.EntityName + " WHERE uuid = $1"
	_, errDelete := db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}

	errDelAcc := asql.base.Del(account)
	if errDelAcc != nil {
		return errDelAcc
	}

	referenceExists := true
	ref, errGet := asql.baseReference.Get(account.Username)
	if errGet != nil {
		if errGet == redis.Nil {
			referenceExists = false
		} else {
			return errGet
		}
	}
	if referenceExists {
		err := asql.baseReference.Del(ref)
		if err != nil {
			return err
		}
	}

	asql.sortedAccount.RemoveItem(account, nil)
	return nil
}

func (asql *AccountRepository) SeedAllAccount() error {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " + asql.app.EntityName
	return asql.sortedAccountSeeder.Seed(query, accountRowsScanner, nil, nil)
}

func accountRowsScanner(rows *sql.Rows) (*model.Account, error) {
	account := model.New()
	err := rows.Scan(
		&account.UUID,
		&account.RandId,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
	)
	return account, err
}

func (asql *AccountRepository) FindByUsername(username string) (*model.Account, error) {
	return AccountRowScanner(asql.findByUsernameStmt.QueryRow(username))
}

func (asql *AccountRepository) SeedByUsername(username string) error {
	account, err := asql.FindByUsername(username)
	if err != nil {
		return err
	}
	if account == nil {
		errSetBlank := asql.baseReference.SetBlank(username)
		if errSetBlank != nil {
			return errSetBlank
		}

		return account2.NotFound
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountRepository) FindByRandId(randId string) (*model.Account, error) {
	return AccountRowScanner(asql.findByRandIdStmt.QueryRow(randId))
}

func (asql *AccountRepository) SeedByRandId(randId string) error {
	account, err := asql.FindByRandId(randId)
	if err != nil {
		if errors.Is(err, account2.NotFound) {
			errSetBlank := asql.baseReference.SetBlank(randId)
			if errSetBlank != nil {
				return errSetBlank
			}
		}
		return err
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func (asql *AccountRepository) FindByEmail(email string) (*model.Account, error) {
	return AccountRowScanner(asql.findByEmailStmt.QueryRow(email))
}

func (asql *AccountRepository) SeedByEmail(email string) error {
	account, err := asql.FindByEmail(email)
	if err != nil {
		return err
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func (asql *AccountRepository) FindByUUID(uuid string) (*model.Account, error) {
	return AccountRowScanner(asql.findByUUIDStmt.QueryRow(uuid))
}

func (asql *AccountRepository) SeedByUUID(uuid string) error {
	account, err := asql.FindByUUID(uuid)
	if err != nil {
		return err
	}

	errSetAcc := asql.base.Set(account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func AccountRowScanner(row *sql.Row) (*model.Account, error) {
	account := model.New()
	err := row.Scan(
		&account.UUID,
		&account.RandId,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
		&account.Base.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, account2.NotFound
		}
		return nil, err
	}

	return account, nil
}

func NewRepository(readDB *sql.DB, redis redis.UniversalClient, app *config.App) *AccountRepository {
	base := redifu.NewBase[*model.Account](redis, app.EntityName+":%s", app.RecordAge)
	baseReference := redifu.NewBase[*model.AccountReference](redis, app.EntityName+":username:%s", app.RecordAge)
	sortedAccount := redifu.NewSorted[*model.Account](redis, base, "account", app.PaginationAge)
	sortedAccountSeeder := redifu.NewSortedSeeder[*model.Account](readDB, base, sortedAccount)

	var errPrepare error
	findByUsernameStmt, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE username = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByRandId, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE randId = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByEmailStmt, errPrepare := readDB.Prepare("" +
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
		app.EntityName + " WHERE email = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByUUIDStmt, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE uuid = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &AccountRepository{
		base:                base,
		baseReference:       baseReference,
		sortedAccount:       sortedAccount,
		sortedAccountSeeder: sortedAccountSeeder,
		redis:               redis,
		findByUsernameStmt:  findByUsernameStmt,
		findByRandIdStmt:    findByRandId,
		findByEmailStmt:     findByEmailStmt,
		findByUUIDStmt:      findByUUIDStmt,
		app:                 app,
	}
}
