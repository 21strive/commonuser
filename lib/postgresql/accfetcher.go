package postgresql

import (
	"context"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type AccountFetchers struct {
	redis      redis.UniversalClient
	base       *redifu.Base[AccountSQL]
	entityName string
}

func (af *AccountFetchers) FetchByUsername(username string) (*AccountSQL, error) {
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

func (af *AccountFetchers) FetchByRandId(randId string) (*AccountSQL, error) {
	account, err := af.base.Get(randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func NewAccountFetchers(redis redis.UniversalClient, entityName string) *AccountFetchers {
	base := redifu.NewBase[AccountSQL](redis, entityName+":%s")
	return &AccountFetchers{
		redis:      redis,
		base:       base,
		entityName: entityName,
	}
}
