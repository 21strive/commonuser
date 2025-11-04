package session

import (
	"github.com/21strive/commonuser/config"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type SessionFetcher struct {
	base *redifu.Base[Session]
}

func (sf *SessionFetcher) FetchByRandId(randId string) (*Session, error) {
	session, err := sf.base.Get(randId)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func NewSessionFetcher(client redis.UniversalClient, app *config.App) *SessionFetcher {
	base := redifu.NewBase[Session](client, app.EntityName+":session:%s", app.RecordAge)
	return &SessionFetcher{
		base: base,
	}
}
