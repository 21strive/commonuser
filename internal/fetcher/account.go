package fetcher

import (
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"

	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type Fetcher struct {
	redis         redis.UniversalClient
	base          *redifu.Base[model.Account]
	baseReference *redifu.Base[model.AccountReference]
	sortedAccount *redifu.Sorted[model.Account]
	entityName    string
}

func (af *Fetcher) Base() *redifu.Base[model.Account] {
	return af.base
}

func (af *Fetcher) FetchByUsername(username string) (*model.Account, error) {
	accountRef, errGetRef := af.baseReference.Get(username)
	if errGetRef != nil {
		return nil, errGetRef
	}
	if accountRef.AccountRandId == "" {
		return nil, nil
	}

	account, err := af.base.Get(accountRef.AccountRandId)
	if err != nil {
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

func (af *Fetcher) FetchByRandId(randId string) (*model.Account, error) {
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

func (af *Fetcher) FetchAll(sortDir string) ([]model.Account, error) {
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
	base := redifu.NewBase[model.Account](redis, app.EntityName+":%s", app.RecordAge)
	sortedAccount := redifu.NewSorted[model.Account](redis, base, "account", app.PaginationAge)
	return &Fetcher{
		redis:         redis,
		base:          base,
		sortedAccount: sortedAccount,
		entityName:    app.EntityName,
	}
}
