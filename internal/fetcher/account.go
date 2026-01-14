package fetcher

import (
	"context"
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
	entityName    string
}

func (af *Fetcher) Base() *redifu.Base[model.Account] {
	return af.base
}

func (af *Fetcher) FetchByUsername(ctx context.Context, username string) (*model.Account, error) {
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

func (af *Fetcher) IsReferenceMissing(ctx context.Context, username string) (bool, error) {
	return af.baseReference.IsMissing(ctx, username)
}

func (af *Fetcher) ReferenceExists(ctx context.Context, username string) error {
	return af.baseReference.Exists(ctx, username)
}

func (af *Fetcher) FetchByRandId(ctx context.Context, randId string) (*model.Account, error) {
	account, err := af.base.Get(ctx, randId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (af *Fetcher) IsAccountMissing(ctx context.Context, randId string) (bool, error) {
	return af.base.IsMissing(ctx, randId)
}

func (af *Fetcher) AccountExists(ctx context.Context, randId string) error {
	return af.base.Exists(ctx, randId)
}

func NewAccountFetchers(redis redis.UniversalClient, app *config.App) *Fetcher {
	base := redifu.NewBase[model.Account](redis, app.EntityName+":%s", app.RecordAge)
	return &Fetcher{
		redis:      redis,
		base:       base,
		entityName: app.EntityName,
	}
}
