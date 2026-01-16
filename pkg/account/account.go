package account

import (
	"context"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
)

type AccountOps struct {
	accountRepository      *repository.AccountRepository
	providerRepository     *repository.ProviderRepository
	verificationRepository *repository.VerificationRepository
	AccountFetcher         *fetcher.AccountFetcher
}

func (o *AccountOps) Init(accountRepository *repository.AccountRepository, providerRepository *repository.ProviderRepository, verificationRepository *repository.VerificationRepository, accountFetcher *fetcher.AccountFetcher) {
	o.accountRepository = accountRepository
	o.providerRepository = providerRepository
	o.verificationRepository = verificationRepository
	o.AccountFetcher = accountFetcher
}

func (o *AccountOps) RegisterWithProvider(ctx context.Context, db database.SQLExecutor, newAccount *model.Account, newProvider *model.Provider) error {
	errCreateProvider := o.providerRepository.Create(ctx, db, newProvider)
	if errCreateProvider != nil {
		return errCreateProvider
	}

	_, errRegister := o.Register(ctx, db, newAccount, false)
	return errRegister
}

func (o *AccountOps) Register(ctx context.Context, db database.SQLExecutor, newAccount *model.Account, requireVerification bool) (*string, error) {
	if !requireVerification {
		newAccount.SetEmailVerified()
	}

	errCreateAcc := o.accountRepository.Create(ctx, db, newAccount)
	if errCreateAcc != nil {
		return nil, errCreateAcc
	}

	var verificationCode string
	var newVerification *model.Verification
	if requireVerification {
		newVerification = model.NewVerification()
		newVerification.SetAccount(newAccount)
		verificationCode = newVerification.SetCode()
		errCreateVerification := o.verificationRepository.Create(ctx, db, newVerification)
		if errCreateVerification != nil {
			return nil, errCreateVerification
		}
	}

	return &verificationCode, nil
}

func (o *AccountOps) Update(ctx context.Context, db database.SQLExecutor, accountUUID string, opt UpdateOpt) (*model.Account, error) {
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

	errSet := o.accountRepository.Update(ctx, db, accountFromDB)
	if errSet != nil {
		return nil, errSet
	}

	errUpdateRef := o.accountRepository.UpdateReference(ctx, accountFromDB, oldUsername, accountFromDB.Username)
	if errUpdateRef != nil {
		return nil, errUpdateRef
	}

	return accountFromDB, nil
}

func (o *AccountOps) Fetch() *AccountFetchers {
	return &AccountFetchers{o: o}
}

func (o *AccountOps) Find() *AccountFinder {
	return &AccountFinder{o: o}
}

func (o *AccountOps) Delete(ctx context.Context, db database.SQLExecutor, account *model.Account) error {
	errDel := o.accountRepository.Delete(ctx, db, account)
	if errDel != nil {
		return errDel
	}

	return nil
}

type UpdateOpt struct {
	NewName     string
	NewUsername string
	NewAvatar   string
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

func (af *AccountFetchers) ByUsername(ctx context.Context, username string) (*model.Account, error) {
	accountFromDB, err := af.o.AccountFetcher.FetchByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) ByRandId(ctx context.Context, randId string) (*model.Account, error) {
	accountFromDB, err := af.o.AccountFetcher.FetchByRandId(ctx, randId)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, model.AccountSeedRequired
	}

	return accountFromDB, nil
}

func New() *AccountOps {
	return &AccountOps{}
}
