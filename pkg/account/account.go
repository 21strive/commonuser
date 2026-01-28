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
	config             *config.App

	Authenticate *Authentication
	Find         *Find
	Fetch        *Fetch
}

func (o *AccountOps) New() *model.Account {
	return model.NewAccount()
}

func (o *AccountOps) SetWriteDB(db *sql.DB) {
	o.writeDB = db
	o.Authenticate.SetWriteDB(db)
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

type Find struct {
	accountRepository *repository.AccountRepository
}

func (af *Find) ByUsername(username string) (*model.Account, error) {
	return af.accountRepository.FindByUsername(username)
}

func (af *Find) ByRandId(randId string) (*model.Account, error) {
	return af.accountRepository.FindByRandId(randId)
}

func (af *Find) ByUUID(uuid string) (*model.Account, error) {
	return af.accountRepository.FindByUUID(uuid)
}

func (af *Find) ByEmail(email string) (*model.Account, error) {
	return af.accountRepository.FindByEmail(email)
}

type Fetch struct {
	accountFetcher *fetcher.AccountFetcher
}

func (af *Fetch) ByUsername(ctx context.Context, username string) (*model.Account, error) {
	accountFromDB, err := af.accountFetcher.FetchByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return accountFromDB, nil
}

func (af *Fetch) ByRandId(ctx context.Context, randId string) (*model.Account, error) {
	accountFromDB, err := af.accountFetcher.FetchByRandId(ctx, randId)
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

func (aup *AuthenticationWithPipe) ByProvider(ctx context.Context, issuer string, sub string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return aup.authOps.byProvider(ctx, aup.pipeline, aup.tx, issuer, sub, deviceInfo)
}

func (aup *AuthenticationWithPipe) ByUsername(ctx context.Context, username string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return aup.authOps.byUsername(ctx, aup.pipeline, aup.tx, username, password, deviceInfo)
}

func (aup *AuthenticationWithPipe) ByEmail(ctx context.Context, email string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return aup.authOps.byEmail(ctx, aup.pipeline, aup.tx, email, password, deviceInfo)
}

// TODO: AuthenticationByTransaction belum ke isi

type Authentication struct {
	writeDB            *sql.DB
	accountRepository  *repository.AccountRepository
	providerRepository *repository.ProviderRepository
	sessionOps         *session.SessionOps
	config             *config.App
}

func (au *Authentication) SetWriteDB(db *sql.DB) {
	au.writeDB = db
}

func (au *Authentication) byProvider(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, issuer string, sub string, deviceInfo *model.DeviceInfo) (string, string, error) {
	providerFromDB, errFind := au.providerRepository.Find(sub, issuer)
	if errFind != nil {
		return "", "", errFind
	}

	accountFromDB, errFind := au.accountRepository.FindByUUID(providerFromDB.AccountUUID)
	if errFind != nil {
		return "", "", errFind
	}

	return au.generateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authentication) ByProvider(ctx context.Context, issuer string, sub string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return au.byProvider(ctx, nil, au.writeDB, issuer, sub, deviceInfo)
}

func (au *Authentication) byUsername(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, username string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.authenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (au *Authentication) ByUsername(ctx context.Context, db types.SQLExecutor, username string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return au.byUsername(ctx, nil, db, username, password, deviceInfo)
}

func (au *Authentication) byEmail(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, email string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.authenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (au *Authentication) ByEmail(ctx context.Context, email string, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	return au.byEmail(ctx, nil, au.writeDB, email, password, deviceInfo)
}

func (au *Authentication) authenticatePassword(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, password string, deviceInfo *model.DeviceInfo) (string, string, error) {
	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return "", "", errVerifyPassword
	}
	if !isAuthenticated {
		return "", "", model.Unauthorized
	}

	return au.generateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authentication) generateToken(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, deviceId string, deviceType string, userAgent string) (string, string, error) {
	session := model.NewSession()
	session.SetDeviceId(deviceId)
	session.SetDeviceType(deviceType)
	session.SetUserAgent(userAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(au.config.TokenLifespan)
	errGenerateToken := session.GenerateRefreshToken()
	if errGenerateToken != nil {
		return "", "", errGenerateToken
	}

	var errCreateSession error
	if pipe != nil {
		errCreateSession = au.sessionOps.WithTransaction(pipe, db.(*sql.Tx)).Create(ctx, session)
	} else {
		errCreateSession = au.sessionOps.Create(ctx, session)
	}
	if errCreateSession != nil {
		return "", "", errCreateSession
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		au.config.JWTSecret,
		au.config.JWTIssuer,
		au.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return "", "", errGenerateAccToken
	}

	return accessToken, session.RefreshToken, nil
}

func (au *Authentication) WithTransaction(pipe redis.Pipeliner, db *sql.Tx) *AuthenticationWithPipe {
	return &AuthenticationWithPipe{authOps: au, pipeline: pipe, tx: db}
}

func New(accountRepository *repository.AccountRepository, providerRepository *repository.ProviderRepository, accountFetcher *fetcher.AccountFetcher, sessionOps *session.SessionOps, config *config.App) *AccountOps {
	authenticate := &Authentication{
		accountRepository:  accountRepository,
		providerRepository: providerRepository,
		sessionOps:         sessionOps,
		config:             config,
	}
	accountFinder := &Find{accountRepository: accountRepository}
	accountFetchers := &Fetch{accountFetcher: accountFetcher}

	return &AccountOps{
		accountRepository:  accountRepository,
		providerRepository: providerRepository,
		accountFetcher:     accountFetcher,
		config:             config,

		Authenticate: authenticate,
		Find:         accountFinder,
		Fetch:        accountFetchers,
	}
}
