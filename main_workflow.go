package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/session"
	"github.com/21strive/commonuser/shared"
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

type Command struct {
	accountRepository *account.Repository
	sessionRepository *session.Repository
	config            *config.App
}

func (aw *Command) AccountBase() *redifu.Base[*account.Account] {
	return aw.accountRepository.GetBase()
}

func (aw *Command) SessionBase() *redifu.Base[*session.Session] {
	return aw.sessionRepository.GetBase()
}

type DeviceInfo struct {
	DeviceId   string
	DeviceInfo string
	UserAgent  string
}

func (aw *Command) AuthenticateByUsername(
	db shared.SQLExecutor,
	username string,
	password string,
	deviceInfo DeviceInfo,
) (*string, *string, *WorkflowError) {
	accountFromDB, errFindUser := aw.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return nil, nil, &WorkflowError{Error: errFindUser, Source: "FindUser"}
	}
	if accountFromDB == nil {
		return nil, nil, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	return aw.Authenticate(
		db,
		accountFromDB,
		password,
		deviceInfo.DeviceId,
		deviceInfo.DeviceInfo,
		deviceInfo.UserAgent,
	)
}

func (aw *Command) AuthenticateByEmail(
	db shared.SQLExecutor,
	email string,
	password string,
	deviceInfo DeviceInfo) (*string, *string, *WorkflowError) {
	accountFromDB, errFindUser := aw.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return nil, nil, &WorkflowError{Error: errFindUser, Source: "FindUser"}
	}
	if accountFromDB == nil {
		return nil, nil, &WorkflowError{Error: account.AccountNotFound, Source: "AccountNotFound"}
	}

	return aw.Authenticate(
		db,
		accountFromDB,
		password,
		deviceInfo.DeviceId,
		deviceInfo.DeviceInfo,
		deviceInfo.UserAgent,
	)
}

func (aw *Command) Authenticate(
	db shared.SQLExecutor,
	accountFromDB *account.Account,
	password string,
	deviceId string,
	deviceInfo string,
	userAgent string) (*string, *string, *WorkflowError) {

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

	err := aw.sessionRepository.Create(db, session)
	if err != nil {
		return nil, nil, &WorkflowError{Error: err, Source: "CreateSession"}
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		aw.config.JWTSecret,
		aw.config.JWTIssuer,
		aw.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return nil, nil, &WorkflowError{Error: errGenerateAccToken, Source: "GenerateAccessToken"}
	}

	return &accessToken, &session.RefreshToken, nil
}

func (aw *Command) RefreshToken(
	db shared.SQLExecutor,
	account *account.Account,
	refreshToken string,
) (*string, *string, *WorkflowError) {

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
	errUpdate := aw.sessionRepository.Update(db, session)
	if errUpdate != nil {
		return nil, nil, &WorkflowError{Error: errUpdate, Source: "UpdateRefreshToken"}
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(aw.config.JWTSecret, aw.config.JWTIssuer, aw.config.JWTLifespan, session.GetRandId())
	if errGenerate != nil {
		return nil, nil, &WorkflowError{Error: errGenerate, Source: "GenerateAccessToken"}
	}

	return &newAccessToken, &session.RefreshToken, nil
}

func (aw *Command) Register(db shared.SQLExecutor, newAccount *account.Account) (*account.Account, *WorkflowError) {
	errCreateAcc := aw.accountRepository.Create(db, newAccount)
	if errCreateAcc != nil {
		return nil, &WorkflowError{Error: errCreateAcc, Source: "Create"}
	}

	return newAccount, nil
}

type UpdateOpt struct {
	NewName     string
	NewUsername string
	NewAvatar   string
}

func (aw *Command) Update(db shared.SQLExecutor, accountUUID string, opt UpdateOpt) error {
	accountFromDB, errFind := aw.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	oldUsername := accountFromDB.Username
	if opt.NewName != "" {
		accountFromDB.SetName(opt.NewName)
	}
	if opt.NewUsername != "" {
		accountFromDB.SetUsername(opt.NewUsername)
	}
	if opt.NewAvatar != "" {
		accountFromDB.SetAvatar(opt.NewAvatar)
	}

	errSet := aw.accountRepository.Update(db, accountFromDB)
	if errSet != nil {
		return errSet
	}

	return aw.accountRepository.UpdateReference(accountFromDB, oldUsername, accountFromDB.Username)
}

func (aw *Command) Delete(db shared.SQLExecutor, account *account.Account) *WorkflowError {
	errDel := aw.accountRepository.Delete(db, account)
	if errDel != nil {
		return &WorkflowError{Error: errDel, Source: "Delete"}
	}

	return nil
}

func (aw *Command) PingSession(db shared.SQLExecutor, sessionRandId string) *WorkflowError {
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
	errUpdateSess := aw.sessionRepository.Update(db, session)
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
	return af.aw.accountRepository.FindByUsername(username)
}

func (af *AccountFinder) ByRandId(randId string) (*account.Account, error) {
	return af.aw.accountRepository.FindByRandId(randId)
}

func (af *AccountFinder) ByUUID(uuid string) (*account.Account, error) {
	return af.aw.accountRepository.FindByUUID(uuid)
}

func (af *AccountFinder) ByEmail(email string) (*account.Account, error) {
	return af.aw.accountRepository.FindByEmail(email)
}

func (aw *Command) Find() *AccountFinder {
	return &AccountFinder{aw: aw}
}

func New(readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Command {
	accountManager := account.NewRepository(readDB, redisClient, app)
	sessionManager := session.NewRepository(readDB, redisClient, app)

	return &Command{
		accountRepository: accountManager,
		sessionRepository: sessionManager,
		config:            app,
	}
}
