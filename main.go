package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/pkg/account"
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

type App struct {
	accountOps      *account.AccountOps
	sessionOps      *session.SessionOps
	verificationOps *verification.VerificationOps
	emailOps        *email.EmailOps
	passwordOps     *password.PasswordOps

	config *config.App
}

func (s *App) WithWriteDB(writeDB *sql.DB) {
	s.accountOps.SetWriteDB(writeDB)
}

func (s *App) AccountBase() *redifu.Base[*model.Account] {
	return s.accountOps.GetAccountBase()
}

func (s *App) SessionBase() *redifu.Base[*model.Session] {
	return s.sessionOps.GetSessionBase()
}

func (s *App) Config() *config.App {
	return s.config
}

func (s *App) Account() *account.AccountOps {
	return s.accountOps
}

func (s *App) Session() *session.SessionOps {
	return s.sessionOps
}

func (s *App) Verification() *verification.VerificationOps {
	return s.verificationOps
}

func (s *App) Email() *email.EmailOps {
	return s.emailOps
}

func (s *App) Password() *password.PasswordOps {
	return s.passwordOps
}

func New(readConnection *sql.DB, redisClient redis.UniversalClient, config *config.App) *App {
	baseAccount := redifu.NewBase[*model.Account](redisClient, config.EntityName+":%s", config.RecordAge)
	baseAccountReference := redifu.NewBase[*model.AccountReference](redisClient, config.EntityName+":username:%s", config.RecordAge)
	baseSession := redifu.NewBase[*model.Session](redisClient, config.EntityName+":session:%s", config.TokenLifespan)

	accountRep := repository.NewAccountRepository(readConnection, redisClient, baseAccount, baseAccountReference, config)
	providerRep := repository.NewProviderRepository(readConnection, config)
	verificationRep := repository.NewVerificationRepository(readConnection, config)
	sessionRep := repository.NewSessionRepository(readConnection, redisClient, baseSession, config)
	updateEmailRep := repository.NewUpdateEmailManager(readConnection, config)
	resetPasswordRep := repository.NewResetPasswordRepository(readConnection, config)

	accountFetcher := fetcher.NewAccountFetchers(redisClient, baseAccount, baseAccountReference, config)
	sessionFetcher := fetcher.NewSessionFetcher(baseSession)

	sessionOps := session.New(sessionRep, sessionFetcher, config)
	accountOps := account.New(accountRep, providerRep, accountFetcher, sessionOps, config)
	verificationOps := verification.New(verificationRep, accountOps, config)
	emailOps := email.New(updateEmailRep, accountOps, sessionOps)
	passwordOps := password.New(resetPasswordRep, sessionOps, accountOps)

	return &App{
		accountOps:      accountOps,
		sessionOps:      sessionOps,
		verificationOps: verificationOps,
		emailOps:        emailOps,
		passwordOps:     passwordOps,
		config:          config,
	}
}
