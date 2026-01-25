package auth

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/redis/go-redis/v9"
	"time"
)

type WithTransaction struct {
	AuthOps  *AuthOps
	Pipeline redis.Pipeliner
	Tx       *sql.Tx
}

func (w *WithTransaction) ByProvider(ctx context.Context, issuer string, sub string, deviceInfo model.DeviceInfo) (string, string, error) {
	return w.AuthOps.byProvider(ctx, w.Pipeline, w.Tx, issuer, sub, deviceInfo)
}

func (w *WithTransaction) ByUsername(ctx context.Context, username string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return w.AuthOps.byUsername(ctx, w.Pipeline, w.Tx, username, password, deviceInfo)
}

func (w *WithTransaction) ByEmail(ctx context.Context, email string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return w.AuthOps.byEmail(ctx, w.Pipeline, w.Tx, email, password, deviceInfo)
}

type AuthOps struct {
	writeDB            *sql.DB
	accountRepository  *repository.AccountRepository
	sessionRepository  *repository.SessionRepository
	providerRepository *repository.ProviderRepository
	config             *config.App
}

func (o *AuthOps) Init(accountRepository *repository.AccountRepository, sessionRepository *repository.SessionRepository, providerRepository *repository.ProviderRepository, config *config.App) {
	o.accountRepository = accountRepository
	o.sessionRepository = sessionRepository
	o.providerRepository = providerRepository
	o.config = config
}

func (o *AuthOps) byProvider(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, issuer string, sub string, deviceInfo model.DeviceInfo) (string, string, error) {
	providerFromDB, errFind := o.providerRepository.Find(sub, issuer)
	if errFind != nil {
		return "", "", errFind
	}

	accountFromDB, errFind := o.accountRepository.FindByUUID(providerFromDB.AccountUUID)
	if errFind != nil {
		return "", "", errFind
	}

	return o.GenerateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (o *AuthOps) ByProvider(ctx context.Context, issuer string, sub string, deviceInfo model.DeviceInfo) (string, string, error) {
	return o.byProvider(ctx, nil, o.writeDB, issuer, sub, deviceInfo)
}

func (o *AuthOps) byUsername(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, username string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := o.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return o.AuthenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (o *AuthOps) ByUsername(ctx context.Context, db types.SQLExecutor, username string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return o.byUsername(ctx, nil, db, username, password, deviceInfo)
}

func (o *AuthOps) byEmail(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, email string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := o.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return o.AuthenticatePassword(ctx, pipe, db, accountFromDB, password, deviceInfo)
}

func (o *AuthOps) ByEmail(ctx context.Context, email string, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	return o.byEmail(ctx, nil, o.writeDB, email, password, deviceInfo)
}

func (o *AuthOps) AuthenticatePassword(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, password string, deviceInfo model.DeviceInfo) (string, string, error) {
	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return "", "", errVerifyPassword
	}
	if !isAuthenticated {
		return "", "", model.Unauthorized
	}

	return o.GenerateToken(ctx, pipe, db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (o *AuthOps) GenerateToken(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountFromDB *model.Account, deviceId string, deviceType string, userAgent string) (string, string, error) {
	session := model.NewSession()
	session.SetDeviceId(deviceId)
	session.SetDeviceType(deviceType)
	session.SetUserAgent(userAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(o.config.TokenLifespan)
	errGenerateToken := session.GenerateRefreshToken()
	if errGenerateToken != nil {
		return "", "", errGenerateToken
	}

	err := o.sessionRepository.Create(ctx, pipe, db, session)
	if err != nil {
		return "", "", err
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		o.config.JWTSecret,
		o.config.JWTIssuer,
		o.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return "", "", errGenerateAccToken
	}

	return accessToken, session.RefreshToken, nil
}

func New(repositoryPool *database.RepositoryPool, appConfig *config.App, writeDB ...*sql.DB) *AuthOps {
	return &AuthOps{
		writeDB:            writeDB[0],
		accountRepository:  repositoryPool.AccountRepository,
		sessionRepository:  repositoryPool.SessionRepository,
		providerRepository: repositoryPool.ProviderRepository,
		config:             appConfig,
	}
}
