package fetcher

import (
	"context"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
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

func NewSessionFetcher(baseSession *redifu.Base[*model.Session]) *SessionFetcher {
	return &SessionFetcher{
		base: baseSession,
	}
}
