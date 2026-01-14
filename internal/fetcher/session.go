package fetcher

import (
	"context"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type SessionFetcher struct {
	base *redifu.Base[*model.Session]
}

func (sf *SessionFetcher) FetchByRandId(ctx context.Context, randId string) (*model.Session, error) {
	session, err := sf.base.Get(ctx, randId)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func NewSessionFetcher(client redis.UniversalClient, app *config.App) *SessionFetcher {
	base := redifu.NewBase[*model.Session](client, app.EntityName+":session:%s", app.RecordAge)
	return &SessionFetcher{
		base: base,
	}
}
