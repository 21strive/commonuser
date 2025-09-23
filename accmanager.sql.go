package commonuser

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type AccountManagerSQL struct {
	db                  *sql.DB
	redis               redis.UniversalClient
	base                *redifu.Base[AccountSQL]
	sortedAccount       *redifu.Sorted[AccountSQL]
	sortedAccountSeeder *redifu.SortedSQLSeeder[AccountSQL]
	entityName          string
}

func (asql *AccountManagerSQL) Create(account *AccountSQL) error {
	query := "INSERT INTO " + asql.entityName + " (uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	_, errInsert := asql.db.Exec(query, account.GetUUID(), account.GetRandId(), account.GetCreatedAt(), account.GetUpdatedAt(), account.Name, account.Username, account.Password, account.Email, account.Avatar, account.Suspended)
	if errInsert != nil {
		return errInsert
	}

	errSet := asql.setAccount(account)
	if errSet != nil {
		return errSet
	}

	asql.sortedAccount.AddItem(*account, nil)
	return nil
}

func (asql *AccountManagerSQL) Patch(account *AccountSQL) error {
	query := "UPDATE " + asql.entityName + " SET updated_at = $1, name = $2, avatar = $3 WHERE uuid = $4"
	_, errUpdate := asql.db.Exec(query, account.GetUpdatedAt(), account.Name, account.Avatar, account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	errSet := asql.setAccount(account)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (asql *AccountManagerSQL) Delete(account *AccountSQL) error {
	query := "DELETE FROM " + asql.entityName + " WHERE uuid = $1"
	_, errDelete := asql.db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}

	errDel := asql.delAccount(account)
	if errDel != nil {
		return errDel
	}

	asql.sortedAccount.RemoveItem(*account, nil)

	return nil
}

func (asql *AccountManagerSQL) setAccount(account *AccountSQL) error {
	if account.Username != "" {
		key := asql.entityName + ":username:" + account.Username
		setReference := asql.redis.Set(context.TODO(), key, account.GetRandId(), 7*24*time.Hour)
		if setReference.Err() != nil {
			return setReference.Err()
		}
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) delAccount(account *AccountSQL) error {
	if account.Username != "" {
		key := asql.entityName + ":username:" + account.Username
		delReference := asql.redis.Del(context.TODO(), key)
		if delReference.Err() != nil {
			return delReference.Err()
		}
	}

	errDelAcc := asql.base.Del(*account)
	if errDelAcc != nil {
		return errDelAcc
	}
	return nil
}

func (asql *AccountManagerSQL) SeedAllAccount() error {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName
	rows, err := asql.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	return asql.sortedAccountSeeder.SeedAll(query, accountRowsScanner, nil, nil)
}

func accountRowsScanner(rows *sql.Rows) (AccountSQL, error) {
	account := NewAccountSQL()
	err := rows.Scan(
		&account.SQLItem.UUID,
		&account.SQLItem.RandId,
		&account.SQLItem.CreatedAt,
		&account.SQLItem.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
		&account.Base.Suspended,
	)
	return *account, err
}

func (asql *AccountManagerSQL) FindByUsername(username string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE username = $1"
	return findOneAccount(asql.db, query, username)
}

func (asql *AccountManagerSQL) SeedByUsername(username string) error {
	account, err := asql.FindByUsername(username)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccount(account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByRandId(randId string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE randId = $1"
	return findOneAccount(asql.db, query, randId)
}

func (asql *AccountManagerSQL) SeedByRandId(randId string) error {
	account, err := asql.FindByRandId(randId)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccount(account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByEmail(email string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE email = $1"
	return findOneAccount(asql.db, query, email)
}

func (asql *AccountManagerSQL) SeedByEmail(email string) error {
	account, err := asql.FindByEmail(email)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccount(account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountManagerSQL) FindByUUID(uuid string) (*AccountSQL, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE uuid = $1"
	return findOneAccount(asql.db, query, uuid)
}

func (asql *AccountManagerSQL) SeedByUUID(uuid string) error {
	account, err := asql.FindByUUID(uuid)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.setAccount(account)
	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func NewAccountManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *AccountManagerSQL {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s", BaseTTL)
	sortedAccount := redifu.NewSorted[AccountSQL](redis, base, "account", SortedSetTTL)
	sortedAccountSeeder := redifu.NewSortedSQLSeeder[AccountSQL](db, base, sortedAccount)
	return &AccountManagerSQL{
		db:                  db,
		base:                base,
		entityName:          entityName,
		sortedAccount:       sortedAccount,
		sortedAccountSeeder: sortedAccountSeeder,
		redis:               redis,
	}
}

func findOneAccount(db *sql.DB, query string, param string) (*AccountSQL, error) {
	row := db.QueryRow(query, param)
	account := NewAccountSQL()
	err := row.Scan(
		&account.SQLItem.UUID,
		&account.SQLItem.RandId,
		&account.SQLItem.CreatedAt,
		&account.SQLItem.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
		&account.Base.Suspended,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return account, nil
}
