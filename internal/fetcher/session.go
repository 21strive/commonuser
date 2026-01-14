package fetcher

import (
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type SessionFetcher struct {
	base   *redifu.Base[*model.Session]
	sorted *redifu.Sorted[*model.Session]
}

func (sf *SessionFetcher) FetchByRandId(randId string) (*model.Session, error) {
	session, err := sf.base.Get(randId)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (sf *SessionFetcher) FetchByAccount(accountRandId string) ([]*model.Session, error) {
	return sf.sorted.Fetch([]string{accountRandId}, "desc", nil, nil)
}

func (sf *SessionFetcher) IsBlankPage(accountRandId string) (bool, error) {
	return sf.sorted.IsBlankPage([]string{accountRandId})
}

func NewSessionFetcher(client redis.UniversalClient, app *config.App) *SessionFetcher {
	base := redifu.NewBase[*model.Session](client, app.EntityName+":session:%s", app.RecordAge)
	sorted := redifu.NewSorted[*model.Session](client, base, app.EntityName+":session:account:%s", app.PaginationAge)
	return &SessionFetcher{
		base:   base,
		sorted: sorted,
	}
}
