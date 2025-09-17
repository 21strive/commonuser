package commonuser

import (
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type SessionFetcherSQL struct {
	base *redifu.Base[SessionSQL]
}

func (sf *SessionFetcherSQL) FetchByRandId(randId string) (*SessionSQL, error) {
	session, err := sf.base.Get(randId)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func NewSessionFetcherSQL(client redis.UniversalClient, entityName string) *SessionFetcherSQL {
	base := redifu.NewBase[SessionSQL](client, entityName+":session:%s", BaseTTL)
	return &SessionFetcherSQL{
		base: base,
	}
}
