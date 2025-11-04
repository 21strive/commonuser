package commonuser

import (
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/session"
	"github.com/21strive/commonuser/shared"
	"github.com/redis/go-redis/v9"
)

type Fetchers struct {
	AccountFetcher *account.Fetcher
	sessionFetcher *session.SessionFetcher
}

type AccountFetchers struct {
	f *Fetchers
}

func (af *AccountFetchers) ByUsername(username string) (*account.Account, error) {
	isBlank, errGet := af.f.AccountFetcher.IsReferenceBlank(username)
	if errGet != nil {
		if !errors.Is(errGet, redis.Nil) {
			return nil, errGet
		}
	}
	if isBlank {
		return nil, account.NotFound
	}

	accountFromDB, err := af.f.AccountFetcher.FetchByUsername(username)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, account.SeedRequired
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) ByRandId(randId string) (*account.Account, error) {
	isBlank, errGet := af.f.AccountFetcher.IsBlank(randId)
	if errGet != nil {
		if !errors.Is(errGet, redis.Nil) {
			return nil, errGet
		}
	}
	if isBlank {
		return nil, account.NotFound
	}

	accountFromDB, err := af.f.AccountFetcher.FetchByRandId(randId)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, account.SeedRequired
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) All(sortDir string) ([]account.Account, error) {
	isBlank, errCheck := af.f.AccountFetcher.IsSortedBlank()
	if errCheck != nil {
		return nil, errCheck
	}
	if isBlank {
		return nil, nil
	}

	accounts, err := af.f.AccountFetcher.FetchAll(sortDir)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, account.SeedRequired
	}

	return accounts, nil
}

func (f *Fetchers) FetchAccount() *AccountFetchers {
	return &AccountFetchers{f: f}
}

func (f *Fetchers) PingSession(sessionRandId string) (*session.Session, error) {
	sessionFromCache, err := f.sessionFetcher.FetchByRandId(sessionRandId)
	if err != nil {
		return nil, err
	}
	if sessionFromCache == nil {
		return nil, shared.Unauthorized
	}
	if !sessionFromCache.IsValid() {
		return nil, shared.Unauthorized
	}

	return sessionFromCache, nil
}

func NewFetchers(redisClient redis.UniversalClient, app *config.App) *Fetchers {
	accountFetcher := account.NewAccountFetchers(redisClient, app)
	sessionFetcher := session.NewSessionFetcher(redisClient, app)
	return &Fetchers{
		AccountFetcher: accountFetcher,
		sessionFetcher: sessionFetcher,
	}
}
