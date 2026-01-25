package cache

import (
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type FetcherPool struct {
	AccountFetcher *fetcher.AccountFetcher
	SessionFetcher *fetcher.SessionFetcher
}

func NewFetcherPool(redisClient redis.UniversalClient, app *config.App) *FetcherPool {
	return &FetcherPool{
		AccountFetcher: fetcher.NewAccountFetchers(redisClient, app),
		SessionFetcher: fetcher.NewSessionFetcher(redisClient, app),
	}
}

type CachePool struct {
	BaseAccount   *redifu.Base[*model.Account]
	BaseReference *redifu.Base[*model.AccountReference]
	BaseSession   *redifu.Base[*model.Session]
}

func NewCachePool(redisClient redis.UniversalClient, app *config.App) *CachePool {
	return &CachePool{
		BaseAccount:   redifu.NewBase[*model.Account](redisClient, app.EntityName+":%s", app.RecordAge),
		BaseReference: redifu.NewBase[*model.AccountReference](redisClient, app.EntityName+":username:%s", app.RecordAge),
		BaseSession:   redifu.NewBase[*model.Session](redisClient, app.EntityName+":session:%s", app.TokenLifespan),
	}
}
