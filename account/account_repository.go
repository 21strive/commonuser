package account

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

// key := asql.entityName + ":username:" + account.Username

type AccountRepository struct {
	db                  *sql.DB
	redis               redis.UniversalClient
	base                *redifu.Base[Account]
	baseReference       *redifu.Base[AccountReference]
	sortedAccount       *redifu.Sorted[Account]
	sortedAccountSeeder *redifu.SortedSQLSeeder[Account]
	entityName          string
}

func (asql *AccountRepository) Base() *redifu.Base[Account] {
	return asql.base
}

func (asql *AccountRepository) Create(account *Account) error {
	query := "INSERT INTO " + asql.entityName + " (uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
	_, errInsert := asql.db.Exec(query, account.GetUUID(), account.GetRandId(), account.GetCreatedAt(), account.GetUpdatedAt(), account.Name, account.Username, account.Password, account.Email, account.Avatar, account.Suspended)
	if errInsert != nil {
		return errInsert
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(*accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	asql.baseReference.DelBlank(account.Username)
	asql.base.DelBlank(account.GetRandId())
	asql.sortedAccount.AddItem(*account, nil)
	return nil
}

func (asql *AccountRepository) Update(account *Account) error {
	query := "UPDATE " + asql.entityName + " SET updated_at = $1, name = $2, username = $3, avatar = $4 WHERE uuid = $5"
	_, errUpdate := asql.db.Exec(query, account.GetUpdatedAt(), account.Name, account.Username, account.Avatar, account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	return nil
}

func (asql *AccountRepository) UpdateReference(account *Account, oldUsername string, newUsername string) error {
	ref, errGet := asql.baseReference.Get(oldUsername)
	if errGet != nil {
		return errGet
	}

	err := asql.baseReference.Del(ref)
	if err != nil {
		return err
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSet := asql.baseReference.Set(*accountReference, newUsername)
	if errSet != nil {
		return errSet
	}

	return nil
}

func (asql *AccountRepository) Delete(account *Account) error {
	query := "DELETE FROM " + asql.entityName + " WHERE uuid = $1"
	_, errDelete := asql.db.Exec(query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}

	errDelAcc := asql.base.Del(*account)
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

	asql.sortedAccount.RemoveItem(*account, nil)
	return nil
}

func (asql *AccountRepository) SeedAllAccount() error {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName
	rows, err := asql.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	return asql.sortedAccountSeeder.Seed(query, accountRowsScanner, nil, nil)
}

func accountRowsScanner(rows *sql.Rows) (Account, error) {
	account := NewAccount()
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

func (asql *AccountRepository) FindByUsername(username string) (*Account, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE username = $1"
	return asql.findOneAccount(query, username)
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

		return AccountNotFound
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(*accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if errSetAcc != nil {
		return errSetAcc
	}
	return nil
}

func (asql *AccountRepository) FindByRandId(randId string) (*Account, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE randId = $1"
	return asql.findOneAccount(query, randId)
}

func (asql *AccountRepository) SeedByRandId(randId string) error {
	account, err := asql.FindByRandId(randId)
	if err != nil {
		return err
	}
	if account == nil {
		errSetBlank := asql.baseReference.SetBlank(randId)
		if errSetBlank != nil {
			return errSetBlank
		}
		return errors.New("account not found")
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(*accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func (asql *AccountRepository) FindByEmail(email string) (*Account, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE email = $1"
	return asql.findOneAccount(query, email)
}

func (asql *AccountRepository) SeedByEmail(email string) error {
	account, err := asql.FindByEmail(email)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(*accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func (asql *AccountRepository) FindByUUID(uuid string) (*Account, error) {
	query := "SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, suspended FROM " + asql.entityName + " WHERE uuid = $1"
	return asql.findOneAccount(query, uuid)
}

func (asql *AccountRepository) SeedByUUID(uuid string) error {
	account, err := asql.FindByUUID(uuid)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	errSetAcc := asql.base.Set(*account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := NewAccountReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := asql.baseReference.Set(*accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	return nil
}

func (asql *AccountRepository) findOneAccount(query string, param string) (*Account, error) {
	row := asql.db.QueryRow(query, param)
	account := NewAccount()
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

func NewAccountRepository(db *sql.DB, redis redis.UniversalClient, entityName string) *AccountRepository {
	base := redifu.NewBase[Account](redis, entityName+":%s", shared.BaseTTL)
	baseReference := redifu.NewBase[AccountReference](redis, entityName+":username:%s", shared.BaseTTL)
	sortedAccount := redifu.NewSorted[Account](redis, base, "account", shared.SortedSetTTL)
	sortedAccountSeeder := redifu.NewSortedSQLSeeder[Account](db, base, sortedAccount)
	return &AccountRepository{
		db:                  db,
		base:                base,
		baseReference:       baseReference,
		entityName:          entityName,
		sortedAccount:       sortedAccount,
		sortedAccountSeeder: sortedAccountSeeder,
		redis:               redis,
	}
}

type AccountFetchers struct {
	redis         redis.UniversalClient
	base          *redifu.Base[Account]
	baseReference *redifu.Base[AccountReference]
	sortedAccount *redifu.Sorted[Account]
	entityName    string
}

func (af *AccountFetchers) Base() *redifu.Base[Account] {
	return af.base
}

func (af *AccountFetchers) FetchByUsername(username string) (*Account, error) {
	accountRef, errGetRef := af.baseReference.Get(username)
	if errGetRef != nil {
		return nil, errGetRef
	}
	if accountRef.AccountRandId == "" {
		return nil, nil
	}

	account, err := af.base.Get(accountRef.AccountRandId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchers) IsReferenceBlank(username string) (bool, error) {
	return af.baseReference.IsBlank(username)
}

func (af *AccountFetchers) DelBlankReference(username string) error {
	return af.baseReference.DelBlank(username)
}

func (af *AccountFetchers) FetchByRandId(randId string) (*Account, error) {
	account, err := af.base.Get(randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchers) IsBlank(randId string) (bool, error) {
	return af.base.IsBlank(randId)
}

func (af *AccountFetchers) DelBlank(randId string) error {
	return af.base.DelBlank(randId)
}

func (af *AccountFetchers) FetchAll(sortDir string) ([]Account, error) {
	account, err := af.sortedAccount.Fetch(nil, sortDir, nil, nil)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (af *AccountFetchers) IsSortedBlank() (bool, error) {
	return af.sortedAccount.IsBlankPage(nil)
}

func (af *AccountFetchers) DelSortedBlank() error {
	return af.sortedAccount.DelBlankPage(nil)
}

func NewAccountFetchers(redis redis.UniversalClient, entityName string) *AccountFetchers {
	base := redifu.NewBase[Account](redis, entityName+":%s", shared.BaseTTL)
	sortedAccount := redifu.NewSorted[Account](redis, base, "account", shared.SortedSetTTL)
	return &AccountFetchers{
		redis:         redis,
		base:          base,
		sortedAccount: sortedAccount,
		entityName:    entityName,
	}
}
