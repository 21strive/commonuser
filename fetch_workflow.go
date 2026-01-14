package commonuser

import (
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/constant"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/session"
	"github.com/redis/go-redis/v9"
)

type Fetchers struct {
	AccountFetcher *fetcher.Fetcher
	sessionFetcher *fetcher.SessionFetcher
}

type SessionFetchers struct {
	f *Fetchers
}

func (sf *SessionFetchers) Ping(sessionRandId string) (*model.Session, error) {
	sessionFromCache, err := sf.f.sessionFetcher.FetchByRandId(sessionRandId)
	if err != nil {
		return nil, err
	}
	if sessionFromCache == nil {
		return nil, constant.Unauthorized
	}
	if !sessionFromCache.IsValid() {
		return nil, constant.Unauthorized
	}

	return sessionFromCache, nil
}

func (sf *SessionFetchers) FetchByAccount(accountRandId string) ([]*model.Session, error) {
	isBlank, errCheck := sf.f.sessionFetcher.IsBlankPage(accountRandId)
	if errCheck != nil {
		return nil, errCheck
	}
	if isBlank {
		return nil, nil
	}

	sessions, err := sf.f.sessionFetcher.FetchByAccount(accountRandId)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, session.SeedRequired
	}

	return sessions, nil
}

func (f *Fetchers) Session() *SessionFetchers {
	return &SessionFetchers{f: f}
}

func NewFetchers(redisClient redis.UniversalClient, app *config.App) *Fetchers {
	accountFetcher := fetcher.NewAccountFetchers(redisClient, app)
	sessionFetcher := fetcher.NewSessionFetcher(redisClient, app)
	return &Fetchers{
		AccountFetcher: accountFetcher,
		sessionFetcher: sessionFetcher,
	}
}
