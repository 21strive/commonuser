package database

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/cache"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/redis/go-redis/v9"
)

type RepositoryPool struct {
	AccountRepository       *repository.AccountRepository
	ProviderRepository      *repository.ProviderRepository
	VerificationRepository  *repository.VerificationRepository
	SessionRepository       *repository.SessionRepository
	UpdateEmailRepository   *repository.UpdateEmailRepository
	ResetPasswordRepository *repository.ResetPasswordRepository
}

func NewRepositoryPool(readDB *sql.DB, redisClient redis.UniversalClient, cachePool *cache.CachePool, appConfig *config.App) *RepositoryPool {
	accountRep := repository.NewAccountRepository(readDB, redisClient, cachePool, appConfig)
	providerRep := repository.NewProviderRepository(readDB, appConfig)
	verificationRep := repository.NewVerificationRepository(readDB, appConfig)
	sessionRep := repository.NewSessionRepository(readDB, redisClient, cachePool, appConfig)
	updateEmailRep := repository.NewUpdateEmailManager(readDB, appConfig)
	resetPasswordRep := repository.NewResetPasswordRepository(readDB, appConfig)

	return &RepositoryPool{
		AccountRepository:       accountRep,
		ProviderRepository:      providerRep,
		VerificationRepository:  verificationRep,
		SessionRepository:       sessionRep,
		UpdateEmailRepository:   updateEmailRep,
		ResetPasswordRepository: resetPasswordRep,
	}
}
