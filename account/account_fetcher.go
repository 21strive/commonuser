package account

import (
	"errors"
	"github.com/21strive/commonuser/config"

	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type Fetcher struct {
	redis         redis.UniversalClient
	base          *redifu.Base[Account]
	baseReference *redifu.Base[AccountReference]
	sortedAccount *redifu.Sorted[Account]
	entityName    string
}

func (af *Fetcher) Base() *redifu.Base[Account] {
	return af.base
}

func (af *Fetcher) FetchByUsername(username string) (*Account, error) {
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

func (af *Fetcher) IsReferenceBlank(username string) (bool, error) {
	return af.baseReference.IsBlank(username)
}

func (af *Fetcher) DelBlankReference(username string) error {
	return af.baseReference.DelBlank(username)
}

func (af *Fetcher) FetchByRandId(randId string) (*Account, error) {
	account, err := af.base.Get(randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *Fetcher) IsBlank(randId string) (bool, error) {
	return af.base.IsBlank(randId)
}

func (af *Fetcher) DelBlank(randId string) error {
	return af.base.DelBlank(randId)
}

func (af *Fetcher) FetchAll(sortDir string) ([]Account, error) {
	account, err := af.sortedAccount.Fetch(nil, sortDir, nil, nil)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (af *Fetcher) IsSortedBlank() (bool, error) {
	return af.sortedAccount.IsBlankPage(nil)
}

func (af *Fetcher) DelSortedBlank() error {
	return af.sortedAccount.DelBlankPage(nil)
}

func NewAccountFetchers(redis redis.UniversalClient, app *config.App) *Fetcher {
	base := redifu.NewBase[Account](redis, app.EntityName+":%s", app.RecordAge)
	sortedAccount := redifu.NewSorted[Account](redis, base, "account", app.PaginationAge)
	return &Fetcher{
		redis:         redis,
		base:          base,
		sortedAccount: sortedAccount,
		entityName:    app.EntityName,
	}
}
