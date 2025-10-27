package commonuser

import (
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/session"
	"github.com/redis/go-redis/v9"
)

type Fetchers struct {
	AccountFetcher *account.Fetcher
	sessionFetcher *session.SessionFetcher
}

func (af *Fetchers) FetchByUsername(username string) (*account.Account, bool, *WorkflowError) {
	accountFromDB, err := af.AccountFetcher.FetchByUsername(username)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchByUsername"}
	}
	if accountFromDB == nil {
		isBlank, errGet := af.AccountFetcher.IsReferenceBlank(username)
		if errGet != nil {
			return nil, false, &WorkflowError{Error: errGet, Source: "IsReferenceBlank"}
		}
		if isBlank {
			return nil, false, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
		}
		return nil, true, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	af.AccountFetcher.DelBlankReference(username)
	af.AccountFetcher.DelBlank(accountFromDB.GetRandId())

	return accountFromDB, false, nil
}

func (af *Fetchers) FetchByRandId(randId string) (*account.Account, bool, *WorkflowError) {
	accountFromDB, err := af.AccountFetcher.FetchByRandId(randId)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchByRandId"}
	}
	if accountFromDB == nil {
		isBlank, errGet := af.AccountFetcher.IsBlank(randId)
		if errGet != nil {
			return nil, false, &WorkflowError{Error: errGet, Source: "IsBlank"}
		}
		if isBlank {
			return nil, false, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
		}
		return nil, true, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	af.AccountFetcher.DelBlank(accountFromDB.GetRandId())
	af.AccountFetcher.DelBlankReference(accountFromDB.Username)

	return accountFromDB, false, nil
}

func (af *Fetchers) FetchAll(sortDir string) ([]account.Account, bool, *WorkflowError) {
	accounts, err := af.AccountFetcher.FetchAll(sortDir)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchAll"}
	}
	if len(accounts) == 0 {
		isBlank, errCheck := af.AccountFetcher.IsSortedBlank()
		if errCheck != nil {
			return nil, false, &WorkflowError{Error: errCheck, Source: "IsBlankPage"}
		}
		if isBlank {
			return nil, false, nil
		}
		return nil, true, nil
	}

	return accounts, false, nil
}

func NewFetchers(redisClient redis.UniversalClient, app *config.App) *Fetchers {
	accountFetcher := account.NewAccountFetchers(redisClient, app)
	sessionFetcher := session.NewSessionFetcher(redisClient, app)
	return &Fetchers{
		AccountFetcher: accountFetcher,
		sessionFetcher: sessionFetcher,
	}
}
