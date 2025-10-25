package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/request"
	"github.com/21strive/commonuser/session"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type WorkflowError struct {
	Error  error
	Source string
}

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
	accountRepository *account.AccountRepository
	sessionRepository *session.SessionRepository
	config            *Config
}

func (aw *Command) AccountBase() *redifu.Base[account.Account] {
	return aw.accountRepository.Base()
}

func (aw *Command) SessionBase() *redifu.Base[session.Session] {
	return aw.sessionRepository.Base()
}

func (aw *Command) AuthenticateByUsername(req *request.NativeAuthRequest) (*string, *string, *WorkflowError) {
	accountFromDB, errFindUser := aw.accountRepository.FindByUsername(req.Username)
	if errFindUser != nil {
		return nil, nil, &WorkflowError{Error: errFindUser, Source: "FindUser"}
	}
	if accountFromDB == nil {
		return nil, nil, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	return aw.Authenticate(accountFromDB, req.Password, req.DeviceId, req.DeviceInfo, req.UserAgent)
}

func (aw *Command) AuthenticateByEmail(req *request.NativeAuthByEmailRequest) (*string, *string, *WorkflowError) {
	accountFromDB, errFindUser := aw.accountRepository.FindByEmail(req.Email)
	if errFindUser != nil {
		return nil, nil, &WorkflowError{Error: errFindUser, Source: "FindUser"}
	}
	if accountFromDB == nil {
		return nil, nil, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	return aw.Authenticate(accountFromDB, req.Password, req.DeviceId, req.DeviceInfo, req.UserAgent)
}

func (aw *Command) Authenticate(
	accountFromDB *account.Account, password string, deviceId string,
	deviceInfo string, userAgent string) (*string, *string, *WorkflowError) {

	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return nil, nil, &WorkflowError{Error: errVerifyPassword, Source: "VerifyPassword"}
	}
	if !isAuthenticated {
		return nil, nil, &WorkflowError{Error: shared.Unauthorized, Source: "WrongPassword"}
	}

	session := session.NewSession()

	session.SetDeviceId(deviceId)
	session.SetDeviceInfo(deviceInfo)
	session.SetUserAgent(userAgent)
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

func (aw *Command) RefreshToken(account *account.Account, refreshToken string) (*string, *string, *WorkflowError) {

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

func (aw *Command) Register(reqBody *request.NativeRegistrationRequest) (*account.Account, *WorkflowError) {
	newAccount := account.NewAccount()
	newAccount.SetEmail(reqBody.Email)
	newAccount.SetPassword(reqBody.Password)
	newAccount.SetName(reqBody.Name)

	if reqBody.Username != "" {
		newAccount.SetUsername(reqBody.Username)
	} else {
		randUsername := item.RandId()
		newAccount.SetUsername(randUsername)
	}

	newAccount.SetAvatar(reqBody.Avatar)

	errCreateAcc := aw.accountRepository.Create(newAccount)
	if errCreateAcc != nil {
		return nil, &WorkflowError{Error: errCreateAcc, Source: "Create"}
	}

	return newAccount, nil
}

func (aw *Command) Delete(account *account.Account) *WorkflowError {
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

func (af *AccountFinder) ByUsername(username string) (*account.Account, error) {
	return af.aw.findByUsername(username)
}

func (af *AccountFinder) ByRandId(randId string) (*account.Account, error) {
	return af.aw.findByRandId(randId)
}

func (af *AccountFinder) ByUUID(uuid string) (*account.Account, error) {
	return af.aw.findByUUID(uuid)
}

func (af *AccountFinder) ByEmail(email string) (*account.Account, error) {
	return af.aw.findByEmail(email)
}

func (aw *Command) Find() *AccountFinder {
	return &AccountFinder{aw: aw}
}

func (aw *Command) findByUsername(username string) (*account.Account, error) {
	return aw.accountRepository.FindByUsername(username)
}

func (aw *Command) findByRandId(randId string) (*account.Account, error) {
	return aw.accountRepository.FindByRandId(randId)
}

func (aw *Command) findByUUID(uuid string) (*account.Account, error) {
	return aw.accountRepository.FindByUUID(uuid)
}

func (aw *Command) findByEmail(email string) (*account.Account, error) {
	return aw.accountRepository.FindByEmail(email)
}

func New(db *sql.DB, redisClient redis.UniversalClient, entityName string, accountConfig *Config) *Command {
	accountManager := account.NewAccountRepository(db, redisClient, entityName)
	sessionManager := session.NewSessionRepository(db, redisClient, entityName)

	return &Command{
		accountRepository: accountManager,
		sessionRepository: sessionManager,
		config:            accountConfig,
	}
}
