package account

import (
	"errors"

	"github.com/21strive/commonuser/shared"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

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
