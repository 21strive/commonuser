package fetcher

import (
	"context"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"

	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type AccountFetcher struct {
	redis         redis.UniversalClient
	base          *redifu.Base[model.Account]
	baseReference *redifu.Base[model.AccountReference]
	entityName    string
}

func (af *AccountFetcher) Base() *redifu.Base[model.Account] {
	return af.base
}

func (af *AccountFetcher) FetchByUsername(ctx context.Context, username string) (*model.Account, error) {
	isMissing, errCheck := af.baseReference.IsMissing(ctx, username)
	if errCheck != nil {
		return nil, errCheck
	}
	if isMissing {
		return nil, model.AccountDoesNotExists
	}

	accountRef, errGetRef := af.baseReference.Get(ctx, username)
	if errGetRef != nil {
		return nil, errGetRef
	}
	if accountRef.AccountRandId == "" {
		return nil, nil
	}

	account, err := af.base.Get(ctx, accountRef.AccountRandId)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetcher) FetchByRandId(ctx context.Context, randId string) (*model.Account, error) {
	account, err := af.base.Get(ctx, randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *AccountFetcher) IsAccountMissing(ctx context.Context, randId string) (bool, error) {
	return af.base.IsMissing(ctx, randId)
}

func (af *AccountFetcher) AccountExists(ctx context.Context, randId string) error {
	return af.base.Exists(ctx, randId)
}

func NewAccountFetchers(redis redis.UniversalClient, app *config.App) *AccountFetcher {
	base := redifu.NewBase[model.Account](redis, app.EntityName+":%s", app.RecordAge)
	return &AccountFetcher{
		redis:      redis,
		base:       base,
		entityName: app.EntityName,
	}
}
