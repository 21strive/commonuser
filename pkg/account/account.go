package account

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/21strive/commonuser/pkg/session"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type WithTransaction struct {
	AccountOps *AccountOps
	Pipeline   redis.Pipeliner
	Tx         *sql.Tx
}

func (w *WithTransaction) Register(ctx context.Context, newAccount *model.Account) error {
	return w.AccountOps.register(ctx, w.Pipeline, w.Tx, newAccount)
}

func (w *WithTransaction) RegisterWithProvider(ctx context.Context, newAccount *model.Account, newProvider *model.Provider) error {
	return w.AccountOps.registerWithProvider(ctx, w.Pipeline, w.Tx, newAccount, newProvider)
}

func (w *WithTransaction) Update(ctx context.Context, newAccount *model.Account) error {
	return w.AccountOps.update(ctx, w.Pipeline, w.Tx, newAccount)
}

func (w *WithTransaction) Delete(ctx context.Context, account *model.Account) error {
	return w.AccountOps.delete(ctx, w.Pipeline, w.Tx, account)
}

type AccountOps struct {
	writeDB            *sql.DB
	accountRepository  *repository.AccountRepository
	providerRepository *repository.ProviderRepository
	accountFetcher     *fetcher.AccountFetcher
	sessionOps         *session.SessionOps
	config             *config.App
}

func (o *AccountOps) New() *model.Account {
	return model.NewAccount()
}

func (o *AccountOps) SetWriteDB(db *sql.DB) {
	o.writeDB = db
}

func (o *AccountOps) GetAccountBase() *redifu.Base[*model.Account] {
	return o.accountRepository.GetBase()
}

func (o *AccountOps) WithTransaction(pipe redis.Pipeliner, db *sql.Tx) *WithTransaction {
	return &WithTransaction{AccountOps: o, Pipeline: pipe, Tx: db}
}

func (o *AccountOps) registerWithProvider(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, newAccount *model.Account, newProvider *model.Provider) error {
	errCreateProvider := o.providerRepository.Create(ctx, db, newProvider)
	if errCreateProvider != nil {
		return errCreateProvider
	}

	return o.register(ctx, pipe, db, newAccount)
}

func (o *AccountOps) RegisterWithProvider(ctx context.Context, newAccount *model.Account, newProvider *model.Provider) error {
	return o.registerWithProvider(ctx, nil, o.writeDB, newAccount, newProvider)
}

func (o *AccountOps) register(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, newAccount *model.Account) error {
	return o.accountRepository.Create(ctx, pipe, db, newAccount)
}

func (o *AccountOps) Register(ctx context.Context, newAccount *model.Account) error {
	return o.register(ctx, nil, o.writeDB, newAccount)
}

func (o *AccountOps) update(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	accountFromDB, errFind := o.accountRepository.FindByUUID(account.GetUUID())
	if errFind != nil {
		return errFind
	}

	oldUsername := accountFromDB.Username

	errSet := o.accountRepository.Update(ctx, pipe, db, accountFromDB)
	if errSet != nil {
		return errSet
	}

	if oldUsername != account.Username {
		return o.accountRepository.UpdateReference(ctx, pipe, accountFromDB, oldUsername, accountFromDB.Username)
	}

	return nil
}

func (o *AccountOps) Update(ctx context.Context, account *model.Account) error {
	return o.update(ctx, nil, o.writeDB, account)
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

func (o *AccountOps) Authenticate() *Authentication {
	return &Authentication{accountOps: o}
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

type AuthenticationWithPipe struct {
	authOps  *Authentication
	pipeline redis.Pipeliner
	tx       *sql.Tx
}

type Authentication struct {
	accountOps *AccountOps
}

func (au *Authentication) byProvider(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, issuer string, sub string, deviceInfo model.DeviceInfo) (string, string, error) {
	providerFromDB, errFind := au.accountOps.providerRepository.Find(sub, issuer)
	if errFind != nil {
		return "", "", errFind
	}

	accountFromDB, errFind := au.accountOps.accountRepository.FindByUUID(providerFromDB.AccountUUID)
	if errFind != nil {
		return "", "", errFind
	}

	return au.GenerateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authentication) ByProvider(ctx context.Context, issuer string, sub string, deviceInfo model.DeviceInfo) (string, string, error) {
	return au.byProvider(ctx, nil, au.accountOps.writeDB, issuer, sub, deviceInfo)
}

func (au *Authentication) byUsername(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, username string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.accountOps.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.AuthenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (au *Authentication) ByUsername(ctx context.Context, db types.SQLExecutor, username string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return au.byUsername(ctx, nil, db, username, password, deviceInfo)
}

func (au *Authentication) byEmail(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, email string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.accountOps.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.AuthenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (au *Authentication) ByEmail(ctx context.Context, email string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return au.byEmail(ctx, nil, au.accountOps.writeDB, email, password, deviceInfo)
}

func (au *Authentication) AuthenticatePassword(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return "", "", errVerifyPassword
	}
	if !isAuthenticated {
		return "", "", model.Unauthorized
	}

	return au.GenerateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authentication) GenerateToken(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, deviceId string, deviceType string, userAgent string) (string, string, error) {
	session := model.NewSession()
	session.SetDeviceId(deviceId)
	session.SetDeviceType(deviceType)
	session.SetUserAgent(userAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(au.accountOps.config.TokenLifespan)
	errGenerateToken := session.GenerateRefreshToken()
	if errGenerateToken != nil {
		return "", "", errGenerateToken
	}

	var errCreateSession error
	if pipe != nil {
		errCreateSession = au.accountOps.sessionOps.WithTransaction(db.(*sql.Tx)).Create(ctx, pipe, session)
	} else {
		errCreateSession = au.accountOps.sessionOps.Create(ctx, session)
	}
	if errCreateSession != nil {
		return "", "", errCreateSession
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		au.accountOps.config.JWTSecret,
		au.accountOps.config.JWTIssuer,
		au.accountOps.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return "", "", errGenerateAccToken
	}

	return accessToken, session.RefreshToken, nil
}

func (au *AuthenticationWithPipe) WithTransaction(pipe redis.Pipeliner, db *sql.Tx) *AuthenticationWithPipe {
	return &AuthenticationWithPipe{authOps: au.authOps, pipeline: pipe, tx: db}
}

func New(accountRepository *repository.AccountRepository, providerRepository *repository.ProviderRepository, accountFetcher *fetcher.AccountFetcher, sessionOps *session.SessionOps, config *config.App) *AccountOps {
	return &AccountOps{
		accountRepository:  accountRepository,
		providerRepository: providerRepository,
		accountFetcher:     accountFetcher,
		sessionOps:         sessionOps,
		config:             config,
	}
}
