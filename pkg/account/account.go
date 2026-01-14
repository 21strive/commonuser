package account

import (
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/redis/go-redis/v9"
)

type AccountOps struct {
	accountRepository      *repository.AccountRepository
	providerRepository     *repository.AccountRepository
	verificationRepository *repository.AccountRepository
	AccountFetcher         *fetcher.Fetcher
}

func (o *AccountOps) RegisterWithProvider(db database.SQLExecutor, newAccount *model.Account, newProvider *model.Provider) error {
	errCreateProvider := o.providerRepository.Create(db, newProvider)
	if errCreateProvider != nil {
		return errCreateProvider
	}

	_, errRegister := o.Register(db, newAccount, false)
	return errRegister
}

func (o *AccountOps) Register(db database.SQLExecutor, newAccount *model.Account, requireVerification bool) (*string, error) {
	if !requireVerification {
		newAccount.SetEmailVerified()
	}

	errCreateAcc := o.accountRepository.Create(db, newAccount)
	if errCreateAcc != nil {
		return nil, errCreateAcc
	}

	var verificationCode string
	var newVerification *model.Verification
	if requireVerification {
		newVerification = model.New()
		newVerification.SetAccount(newAccount)
		verificationCode = newVerification.SetCode()
		errCreateVerification := o.verificationRepository.Create(db, newVerification)
		if errCreateVerification != nil {
			return nil, errCreateVerification
		}
	}

	return &verificationCode, nil
}

type UpdateOpt struct {
	NewName     string
	NewUsername string
	NewAvatar   string
}

func (o *AccountOps) Update(db database.SQLExecutor, accountUUID string, opt UpdateOpt) (*model.Account, error) {
	accountFromDB, errFind := o.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	oldUsername := accountFromDB.Username
	if opt.NewName != "" {
		accountFromDB.SetName(opt.NewName)
	}
	if opt.NewUsername != "" {
		accountFromDB.SetUsername(opt.NewUsername)
	}
	if opt.NewAvatar != "" {
		accountFromDB.SetAvatar(opt.NewAvatar)
	}

	errSet := o.accountRepository.Update(db, accountFromDB)
	if errSet != nil {
		return nil, errSet
	}

	errUpdateRef := o.accountRepository.UpdateReference(accountFromDB, oldUsername, accountFromDB.Username)
	if errUpdateRef != nil {
		return nil, errUpdateRef
	}

	return accountFromDB, nil
}

func (o *AccountOps) SeedAccount() error {
	errSeed := o.accountRepository.SeedAllAccount()
	if errSeed != nil {
		return errSeed
	}

	return nil
}

func (o *AccountOps) Find() *AccountFinder {
	return &AccountFinder{o: o}
}

func (o *AccountOps) Delete(db database.SQLExecutor, account *model.Account) error {
	errDel := o.accountRepository.Delete(db, account)
	if errDel != nil {
		return errDel
	}

	return nil
}

type AccountFinder struct {
	o *AccountOps
}

func (af *AccountFinder) ByUsername(username string) (*model.Account, error) {
	return af.o.accountRepository.FindByUsername(username)
}

func (af *AccountFinder) ByRandId(randId string) (*model.Account, error) {
	return af.o.accountRepository.FindByRandId(randId)
}

func (af *AccountFinder) ByUUID(uuid string) (*model.Account, error) {
	return af.o.accountRepository.FindByUUID(uuid)
}

func (af *AccountFinder) ByEmail(email string) (*model.Account, error) {
	return af.o.accountRepository.FindByEmail(email)
}

type AccountFetchers struct {
	o *AccountOps
}

func (af *AccountFetchers) ByUsername(username string) (*model.Account, error) {
	isBlank, errGet := af.AccountFetcher.IsReferenceBlank(username)
	if errGet != nil {
		if !errors.Is(errGet, redis.Nil) {
			return nil, errGet
		}
	}
	if isBlank {
		return nil, account.NotFound
	}

	accountFromDB, err := af.AccountFetcher.FetchByUsername(username)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, account.SeedRequired
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) ByRandId(randId string) (*model.Account, error) {
	isBlank, errGet := af.AccountFetcher.IsBlank(randId)
	if errGet != nil {
		if !errors.Is(errGet, redis.Nil) {
			return nil, errGet
		}
	}
	if isBlank {
		return nil, account.NotFound
	}

	accountFromDB, err := af.AccountFetcher.FetchByRandId(randId)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, account.SeedRequired
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) All(sortDir string) ([]model.Account, error) {
	isBlank, errCheck := af.AccountFetcher.IsSortedBlank()
	if errCheck != nil {
		return nil, errCheck
	}
	if isBlank {
		return nil, nil
	}

	accounts, err := af.AccountFetcher.FetchAll(sortDir)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, account.SeedRequired
	}

	return accounts, nil
}

func (f *Fetchers) Account() *AccountFetchers {
	return &AccountFetchers{o: f}
}
