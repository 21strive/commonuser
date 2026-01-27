package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/cache"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/pkg/account"
	"github.com/21strive/commonuser/pkg/auth"
	"github.com/21strive/commonuser/pkg/email"
	"github.com/21strive/commonuser/pkg/password"
	"github.com/21strive/commonuser/pkg/session"
	"github.com/21strive/commonuser/pkg/verification"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type (
	Account       = model.Account
	Session       = model.Session
	Verification  = model.Verification
	Provider      = model.Provider
	UpdateEmail   = model.UpdateEmail
	ResetPassword = model.ResetPassword
)

func IsAccountNotFound(err error) bool {
	return errors.Is(err, model.AccountDoesNotExists)
}

func IsAccountSeedRequired(err error) bool {
	return errors.Is(err, model.AccountSeedRequired)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, model.Unauthorized)
}

func IsInvalidSession(err error) bool {
	return errors.Is(err, model.InvalidSession)
}

func IsSessionNotFound(err error) bool {
	return errors.Is(err, model.SessionNotFound)
}

func IsProviderNotFound(err error) bool {
	return errors.Is(err, model.ProviderNotFound)
}

func IsVerificationNotFound(err error) bool {
	return errors.Is(err, model.VerificationNotFound)
}

func IsInvalidVerificationCode(err error) bool {
	return errors.Is(err, model.InvalidVerificationCode)
}

func IsRequestExpired(err error) bool {
	return errors.Is(err, model.EmailChangeRequestExpired)
}

func IsInvalidEmailChangeToken(err error) bool {
	return errors.Is(err, model.InvalidEmailChangeToken)
}

func IsEmailChangeTokenNotFound(err error) bool {
	return errors.Is(err, model.EmailChangeTokenNotFound)
}

func IsInvalidResetPasswordToken(err error) bool {
	return errors.Is(err, model.InvalidResetPasswordToken)
}

func IsResetPasswordRequestExpired(err error) bool {
	return errors.Is(err, model.ResetPasswordRequestExpired)
}

func IsResetPasswordTicketNotFound(err error) bool {
	return errors.Is(err, model.ResetPasswordTicketNotFound)
}

type Service struct {
	writeDB        *sql.DB
	cachePool      *cache.CachePool
	fetcherPool    *cache.FetcherPool
	repositoryPool *database.RepositoryPool
	config         *config.App
}

func (s *Service) WithWriteDB(db *sql.DB) {
	s.writeDB = db
}

func (s *Service) AccountBase() *redifu.Base[*model.Account] {
	return s.repositoryPool.AccountRepository.GetBase()
}

func (s *Service) SessionBase() *redifu.Base[*model.Session] {
	return s.repositoryPool.SessionRepository.GetBase()
}

func (s *Service) Config() *config.App {
	return s.config
}

// Builder methods - return operation structs with service reference
func (s *Service) Auth() *auth.AuthOps {
	return auth.New(s.repositoryPool, s.config, s.writeDB)
}

func (s *Service) Account() *account.AccountOps {
	return account.New(s.repositoryPool, s.fetcherPool, s.writeDB)
}

func (s *Service) Session() *session.SessionOps {
	return session.New(s.repositoryPool, s.fetcherPool, s.config, s.writeDB)
}

func (s *Service) Verification() *verification.VerificationOps {
	return verification.New(s.repositoryPool, s.config, s.writeDB)
}

func (s *Service) Email() *email.EmailOps {
	return email.New(s.repositoryPool, s.writeDB)
}

func (s *Service) Password() *password.PasswordOps {
	return password.New(s.repositoryPool, s.writeDB)
}

func New(readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Service {
	cachePool := cache.NewCachePool(redisClient, app)
	repositories := database.NewRepositoryPool(readDB, redisClient, cachePool, app)
	fetchers := cache.NewFetcherPool(redisClient, app)

	return &Service{
		cachePool:      cachePool,
		fetcherPool:    fetchers,
		repositoryPool: repositories,
		config:         app,
	}
}
