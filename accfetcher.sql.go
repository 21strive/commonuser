package commonuser

import (
	"context"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type AccountFetchersSQL struct {
	redis         redis.UniversalClient
	base          *redifu.Base[AccountSQL]
	sortedAccount *redifu.Sorted[AccountSQL]
	entityName    string
}

func (af *AccountFetchersSQL) FetchByUsername(username string) (*AccountSQL, error) {
	getRandID := af.redis.Get(context.TODO(), af.entityName+":username:"+username)
	if getRandID.Err() != nil {
		if getRandID.Err() == redis.Nil {
			return nil, nil
		}
		return nil, getRandID.Err()
	}

	account, err := af.base.Get(getRandID.Val())
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchersSQL) FetchByRandId(randId string) (*AccountSQL, error) {
	account, err := af.base.Get(randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetchersSQL) FetchAll(sortDir string) ([]AccountSQL, error) {
	account, err := af.sortedAccount.Fetch(nil, sortDir, nil, nil)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func NewAccountFetchers(redis redis.UniversalClient, entityName string) *AccountFetchersSQL {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s", BaseTTL)
	sortedAccount := redifu.NewSorted[AccountSQL](redis, base, "account", SortedSetTTL)
	return &AccountFetchersSQL{
		redis:         redis,
		base:          base,
		sortedAccount: sortedAccount,
		entityName:    entityName,
	}
}
