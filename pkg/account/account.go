package account

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/internal/cache"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/redis/go-redis/v9"
)

type WithTransaction struct {
	AccountOps *AccountOps
	Pipeline   redis.Pipeliner
	Tx         *sql.Tx
}

func (w *WithTransaction) Register(ctx context.Context, newAccount *model.Account, requireVerification bool) (*string, error) {
	return w.AccountOps.register(ctx, w.Pipeline, w.Tx, newAccount, requireVerification)
}

func (w *WithTransaction) RegisterWithProvider(ctx context.Context, newAccount *model.Account, newProvider *model.Provider) error {
	return w.AccountOps.registerWithProvider(ctx, w.Pipeline, w.Tx, newAccount, newProvider)
}

func (w *WithTransaction) Update(ctx context.Context, accountUUID string, opt UpdateOpt) (*model.Account, error) {
	return w.AccountOps.update(ctx, w.Pipeline, w.Tx, accountUUID, opt)
}

func (w *WithTransaction) Delete(ctx context.Context, account *model.Account) error {
	return w.AccountOps.delete(ctx, w.Pipeline, w.Tx, account)
}

type AccountOps struct {
	writeDB                *sql.DB
	accountRepository      *repository.AccountRepository
	providerRepository     *repository.ProviderRepository
	verificationRepository *repository.VerificationRepository
	accountFetcher         *fetcher.AccountFetcher
}

func (o *AccountOps) Init(accountRepository *repository.AccountRepository, providerRepository *repository.ProviderRepository, verificationRepository *repository.VerificationRepository, accountFetcher *fetcher.AccountFetcher) {
	o.accountRepository = accountRepository
	o.providerRepository = providerRepository
	o.verificationRepository = verificationRepository
	o.accountFetcher = accountFetcher
}

func (o *AccountOps) WithTransaction(pipe redis.Pipeliner, db *sql.Tx) *WithTransaction {
	return &WithTransaction{AccountOps: o, Pipeline: pipe, Tx: db}
}

func (o *AccountOps) registerWithProvider(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, newAccount *model.Account, newProvider *model.Provider) error {
	errCreateProvider := o.providerRepository.Create(ctx, db, newProvider)
	if errCreateProvider != nil {
		return errCreateProvider
	}

	_, errRegister := o.register(ctx, pipe, db, newAccount, false)
	return errRegister
}

func (o *AccountOps) RegisterWithProvider(ctx context.Context, newAccount *model.Account, newProvider *model.Provider) error {
	return o.registerWithProvider(ctx, nil, o.writeDB, newAccount, newProvider)
}

func (o *AccountOps) register(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, newAccount *model.Account, requireVerification bool) (*string, error) {
	if !requireVerification {
		newAccount.SetEmailVerified()
	}

	errCreateAcc := o.accountRepository.Create(ctx, pipe, db, newAccount)
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

func (o *AccountOps) Register(ctx context.Context, newAccount *model.Account, requireVerification bool) (*string, error) {
	return o.register(ctx, nil, o.writeDB, newAccount, requireVerification)
}

func (o *AccountOps) update(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountUUID string, opt UpdateOpt) (*model.Account, error) {
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

	errSet := o.accountRepository.Update(ctx, pipe, db, accountFromDB)
	if errSet != nil {
		return nil, errSet
	}

	errUpdateRef := o.accountRepository.UpdateReference(ctx, pipe, accountFromDB, oldUsername, accountFromDB.Username)
	if errUpdateRef != nil {
		return nil, errUpdateRef
	}

	return accountFromDB, nil
}

func (o *AccountOps) Update(ctx context.Context, accountUUID string, opt UpdateOpt) (*model.Account, error) {
	return o.update(ctx, nil, o.writeDB, accountUUID, opt)
}

func (o *AccountOps) Fetch() *AccountFetchers {
	return &AccountFetchers{o: o}
}

func (o *AccountOps) Find() *AccountFinder {
	return &AccountFinder{o: o}
}

func (o *AccountOps) delete(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	errDel := o.accountRepository.Delete(ctx, pipe, db, account)
	if errDel != nil {
		return errDel
	}

	return nil
}

func (o *AccountOps) Delete(ctx context.Context, account *model.Account) error {
	return o.delete(ctx, nil, o.writeDB, account)
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
	accountFromDB, err := af.o.accountFetcher.FetchByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return accountFromDB, nil
}

func (af *AccountFetchers) ByRandId(ctx context.Context, randId string) (*model.Account, error) {
	accountFromDB, err := af.o.accountFetcher.FetchByRandId(ctx, randId)
	if err != nil {
		return nil, err
	}
	if accountFromDB == nil {
		return nil, model.AccountSeedRequired
	}

	return accountFromDB, nil
}

func New(repositoryPool *database.RepositoryPool, fetcherPool *cache.FetcherPool, writeDB ...*sql.DB) *AccountOps {
	return &AccountOps{
		writeDB:                writeDB[0],
		accountRepository:      repositoryPool.AccountRepository,
		providerRepository:     repositoryPool.ProviderRepository,
		verificationRepository: repositoryPool.VerificationRepository,
		accountFetcher:         fetcherPool.AccountFetcher,
	}
}
