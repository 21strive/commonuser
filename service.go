package commonuser

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/pkg/account"
	"github.com/21strive/commonuser/pkg/auth"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	accountRepository       *repository.AccountRepository
	sessionRepository       *repository.SessionRepository
	verificationRepository  *repository.VerificationRepository
	updateEmailRepository   *repository.UpdateEmailRepository
	resetPasswordRepository *repository.ResetPasswordRepository
	providerRepository      *repository.ProviderRepository
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
	return &auth.AuthOps{
		accountRepository:  s.accountRepository,
		sessionRepository:  s.sessionRepository,
		providerRepository: s.providerRepository,
		config:             s.config,
	}
}

func (s *Service) Account() *account.Operations {
	return &account.AccountOps{
		accountRepository: s.accountRepository,
		config:            s.config,
	}
}

func (s *Service) Session() *session.Operations {
	return &session.Operations{
		sessionRepository: s.sessionRepository,
		config:            s.config,
	}
}

func (s *Service) Verification() *verification.Operations {
	return &verification.Operations{
		accountRepository:      s.accountRepository,
		verificationRepository: s.verificationRepository,
		config:                 s.config,
	}
}

func (s *Service) Email() *email.Operations {
	return &email.Operations{
		accountRepository:     s.accountRepository,
		updateEmailRepository: s.updateEmailRepository,
		sessionRepository:     s.sessionRepository,
	}
}

func (s *Service) Password() *password.Operations {
	return &password.Operations{
		accountRepository:       s.accountRepository,
		resetPasswordRepository: s.resetPasswordRepository,
		sessionRepository:       s.sessionRepository,
	}
}

func New(writeDB *sql.DB, readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Service {
	accountManager := repository.NewAccountRepository(readDB, redisClient, app)
	sessionManager := repository.NewAccountRepository(readDB, redisClient, app)
	verificationManager := repository.NewAccountRepository(readDB, app)
	updateEmailManager := repository.NewUpdateEmailManager(readDB, app)
	resetPasswordManager := repository.NewAccountRepository(readDB, app)
	providerRepository := repository.NewAccountRepository(readDB, app)

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
