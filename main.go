package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
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
	accountRepository       *repository.AccountRepository
	sessionRepository       *repository.SessionRepository
	verificationRepository  *repository.VerificationRepository
	updateEmailRepository   *repository.UpdateEmailRepository
	resetPasswordRepository *repository.ResetPasswordRepository
	providerRepository      *repository.ProviderRepository
	accountFetcher          *fetcher.AccountFetcher
	sessionFetcher          *fetcher.SessionFetcher
	config                  *config.App
}

func (aw *Service) AccountBase() *redifu.Base[*model.Account] {
	return aw.accountRepository.GetBase()
}

func (aw *Service) SessionBase() *redifu.Base[*model.Session] {
	return aw.sessionRepository.GetBase()
}

func (aw *Service) Config() *config.App {
	return aw.config
}

// Builder methods - return operation structs with service reference
func (s *Service) Auth() *auth.AuthOps {
	auth := auth.New()
	auth.Init(
		s.accountRepository,
		s.sessionRepository,
		s.providerRepository,
		s.config,
	)

	return auth
}

func (s *Service) Account() *account.AccountOps {
	account := account.New()
	account.Init(
		s.accountRepository,
		s.providerRepository,
		s.verificationRepository,
		s.accountFetcher,
	)

	return account
}

func (s *Service) Session() *session.SessionOps {
	session := session.New()
	session.Init(
		s.sessionRepository,
		s.sessionFetcher,
		s.config,
	)

	return session
}

func (s *Service) Verification() *verification.VerificationOps {
	verification := verification.New()
	verification.Init(
		s.accountRepository,
		s.sessionRepository,
		s.providerRepository,
		s.verificationRepository,
		s.config,
	)

	return verification
}

func (s *Service) Email() *email.EmailOps {
	email := email.New()
	email.Init(
		s.updateEmailRepository,
		s.accountRepository,
		s.sessionRepository,
	)

	return email
}

func (s *Service) Password() *password.PasswordOps {
	password := password.New()
	password.Init(
		s.resetPasswordRepository,
		s.sessionRepository,
		s.accountRepository,
	)

	return password
}

func New(readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Service {
	accountManager := repository.NewAccountRepository(readDB, redisClient, app)
	sessionManager := repository.NewSessionRepository(readDB, redisClient, app)
	verificationManager := repository.NewVerificationRepository(readDB, app)
	updateEmailManager := repository.NewUpdateEmailManager(readDB, app)
	resetPasswordManager := repository.NewResetPasswordRepository(readDB, app)
	providerRepository := repository.NewProviderRepository(readDB, app)

	return &Service{
		accountRepository:       accountManager,
		sessionRepository:       sessionManager,
		verificationRepository:  verificationManager,
		updateEmailRepository:   updateEmailManager,
		resetPasswordRepository: resetPasswordManager,
		providerRepository:      providerRepository,
		config:                  app,
	}
}

func NewAccount() *model.Account {
	return model.NewAccount()
}
