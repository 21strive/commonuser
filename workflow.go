package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type WorkflowError struct {
	Error  error
	Source string
}

var AccountNotFound = errors.New("account not found!")
var SessionNotFound = errors.New("session not found!")
var InvalidSession = errors.New("invalid session!")

type Config struct {
	EntityName    string
	TokenLifespan time.Duration
	JWTSecret     string
	JWTIssuer     string
	JWTLifespan   time.Duration
}

type Command struct {
	accountRepository *AccountRepository
	sessionRepository *SessionRepository
	config            *Config
}

func (aw *Command) AccountBase() *redifu.Base[Account] {
	return aw.accountRepository.Base()
}

func (aw *Command) SessionBase() *redifu.Base[Session] {
	return aw.sessionRepository.Base()
}

func (aw *Command) Authenticate(req *NativeAuthRequest) (*string, *string, *WorkflowError) {
	accountFromDB, errFindUser := aw.accountRepository.FindByUsername(req.Username)
	if errFindUser != nil {
		return nil, nil, &WorkflowError{Error: errFindUser, Source: "FindUser"}
	}
	if accountFromDB == nil {
		return nil, nil, &WorkflowError{Error: AccountNotFound, Source: "AccountNotFound"}
	}

	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(req.Password)
	if errVerifyPassword != nil {
		return nil, nil, &WorkflowError{Error: errVerifyPassword, Source: "VerifyPassword"}
	}
	if !isAuthenticated {
		return nil, nil, &WorkflowError{Error: Unauthorized, Source: "WrongPassword"}
	}

	session := NewSession()

	session.SetDeviceId(req.DeviceId)
	session.SetDeviceInfo(req.DeviceInfo)
	session.SetUserAgent(req.UserAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(aw.config.TokenLifespan)
	session.GenerateRefreshToken()

	err := aw.sessionRepository.Create(session)
	if err != nil {
		return nil, nil, &WorkflowError{Error: err, Source: "CreateSession"}
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(aw.config.JWTSecret, aw.config.JWTIssuer, aw.config.JWTLifespan, session.GetRandId())
	if errGenerateAccToken != nil {
		return nil, nil, &WorkflowError{Error: errGenerateAccToken, Source: "GenerateAccessToken"}
	}

	return &accessToken, &session.RefreshToken, nil
}

func (aw *Command) RefreshToken(account *Account, refreshToken string) (*string, *string, *WorkflowError) {

	session, errFind := aw.sessionRepository.FindByHash(refreshToken)
	if errFind != nil {
		return nil, nil, &WorkflowError{Error: errFind, Source: "FindByRefreshToken"}
	}
	if session == nil {
		return nil, nil, &WorkflowError{Error: SessionNotFound, Source: "InvalidSession"}
	}

	valid := session.IsValid()
	if !valid {
		return nil, nil, &WorkflowError{Error: InvalidSession, Source: "InvalidSession"}
	}

	// generate new refresh token
	session.GenerateRefreshToken()
	session.SetLastActiveAt(time.Now().UTC())
	errUpdate := aw.sessionRepository.Update(session)
	if errUpdate != nil {
		return nil, nil, &WorkflowError{Error: errUpdate, Source: "UpdateRefreshToken"}
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(aw.config.JWTSecret, aw.config.JWTIssuer, aw.config.JWTLifespan, session.GetRandId())
	if errGenerate != nil {
		return nil, nil, &WorkflowError{Error: errGenerate, Source: "GenerateAccessToken"}
	}

	return &newAccessToken, &session.RefreshToken, nil
}

func (aw *Command) Register(reqBody *NativeRegistrationRequest) (*Account, *WorkflowError) {
	newAccount := NewAccount()
	newAccount.SetEmail(reqBody.Email)
	newAccount.SetPassword(reqBody.Password)
	newAccount.SetName(reqBody.Name)
	newAccount.SetUsername(reqBody.Username)
	newAccount.SetAvatar(reqBody.Avatar)

	errCreateAcc := aw.accountRepository.Create(newAccount)
	if errCreateAcc != nil {
		return nil, &WorkflowError{Error: errCreateAcc, Source: "Create"}
	}

	return newAccount, nil
}

func (aw *Command) Delete(account *Account) *WorkflowError {
	errDel := aw.accountRepository.Delete(account)
	if errDel != nil {
		return &WorkflowError{Error: errDel, Source: "Delete"}
	}

	return nil
}

func (aw *Command) PingSession(sessionRandId string) *WorkflowError {
	session, errFind := aw.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return &WorkflowError{
			Error: errFind,
		}
	}
	if session == nil {
		return nil
	}

	session.SetLastActiveAt(time.Now().UTC())
	errUpdateSess := aw.sessionRepository.Update(session)
	if errUpdateSess != nil {
		return &WorkflowError{
			Error: errUpdateSess,
		}
	}

	return nil
}

func (aw *Command) SeedAccount() *WorkflowError {
	errSeed := aw.accountRepository.SeedAllAccount()
	if errSeed != nil {
		return &WorkflowError{Error: errSeed, Source: "Seed"}
	}

	return nil
}

type AccountFinder struct {
	aw *Command
}

func (af *AccountFinder) ByUsername(username string) (*Account, error) {
	return af.aw.findByUsername(username)
}

func (af *AccountFinder) ByRandId(randId string) (*Account, error) {
	return af.aw.findByRandId(randId)
}

func (af *AccountFinder) ByUUID(uuid string) (*Account, error) {
	return af.aw.findByUUID(uuid)
}

func (af *AccountFinder) ByEmail(email string) (*Account, error) {
	return af.aw.findByEmail(email)
}

func (aw *Command) Find() *AccountFinder {
	return &AccountFinder{aw: aw}
}

func (aw *Command) findByUsername(username string) (*Account, error) {
	return aw.accountRepository.FindByUsername(username)
}

func (aw *Command) findByRandId(randId string) (*Account, error) {
	return aw.accountRepository.FindByRandId(randId)
}

func (aw *Command) findByUUID(uuid string) (*Account, error) {
	return aw.accountRepository.FindByUUID(uuid)
}

func (aw *Command) findByEmail(email string) (*Account, error) {
	return aw.accountRepository.FindByEmail(email)
}

func New(db *sql.DB, redisClient redis.UniversalClient, entityName string, accountConfig *Config) *Command {
	accountManager := NewAccountRepository(db, redisClient, entityName)
	sessionManager := NewSessionRepository(db, redisClient, entityName)

	return &Command{
		accountRepository: accountManager,
		sessionRepository: sessionManager,
		config:            accountConfig,
	}
}

type Fetchers struct {
	AccountFetcher *AccountFetchers
	sessionFetcher *SessionFetcher
}

func (af *Fetchers) FetchByUsername(username string) (*Account, bool, *WorkflowError) {
	account, err := af.AccountFetcher.FetchByUsername(username)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchByUsername"}
	}
	if account == nil {
		isBlank, errGet := af.AccountFetcher.IsReferenceBlank(username)
		if errGet != nil {
			return nil, false, &WorkflowError{Error: errGet, Source: "IsReferenceBlank"}
		}
		if isBlank {
			return nil, false, &WorkflowError{Error: AccountNotFound, Source: "AccountNotFound"}
		}
		return nil, true, &WorkflowError{Error: AccountNotFound, Source: "AccountNotFound"}
	}

	af.AccountFetcher.DelBlankReference(username)
	af.AccountFetcher.DelBlank(account.GetRandId())

	return account, false, nil
}

func (af *Fetchers) FetchByRandId(randId string) (*Account, bool, *WorkflowError) {
	account, err := af.AccountFetcher.FetchByRandId(randId)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchByRandId"}
	}
	if account == nil {
		isBlank, errGet := af.AccountFetcher.IsBlank(randId)
		if errGet != nil {
			return nil, false, &WorkflowError{Error: errGet, Source: "IsBlank"}
		}
		if isBlank {
			return nil, false, &WorkflowError{Error: AccountNotFound, Source: "AccountNotFound"}
		}
		return nil, true, &WorkflowError{Error: AccountNotFound, Source: "AccountNotFound"}
	}

	af.AccountFetcher.DelBlank(account.GetRandId())
	af.AccountFetcher.DelBlankReference(account.Username)

	return account, false, nil
}

func (af *Fetchers) FetchAll(sortDir string) ([]Account, bool, *WorkflowError) {
	accounts, err := af.AccountFetcher.FetchAll(sortDir)
	if err != nil {
		return nil, false, &WorkflowError{Error: err, Source: "FetchAll"}
	}
	if len(accounts) == 0 {
		isBlank, errCheck := af.AccountFetcher.IsSortedBlank()
		if errCheck != nil {
			return nil, false, &WorkflowError{Error: errCheck, Source: "IsBlankPage"}
		}
		if isBlank {
			return nil, false, nil
		}
		return nil, true, nil
	}

	return accounts, false, nil
}

func NewFetchers(redisClient redis.UniversalClient, entityName string) *Fetchers {
	accountFetcher := NewAccountFetchers(redisClient, entityName)
	sessionFetcher := NewSessionFetcher(redisClient, entityName)
	return &Fetchers{
		AccountFetcher: accountFetcher,
		sessionFetcher: sessionFetcher,
	}
}
