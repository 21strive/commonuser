package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/session"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/commonuser/verification"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

var SessionNotFound = errors.New("session not found!")
var InvalidSession = errors.New("invalid session!")

type Service struct {
	accountRepository      *account.Repository
	sessionRepository      *session.Repository
	verificationRepository *verification.Repository
	config                 *config.App
}

func (aw *Service) AccountBase() *redifu.Base[*account.Account] {
	return aw.accountRepository.GetBase()
}

func (aw *Service) SessionBase() *redifu.Base[*session.Session] {
	return aw.sessionRepository.GetBase()
}

type DeviceInfo struct {
	DeviceId   string
	DeviceInfo string
	UserAgent  string
}

func (aw *Service) AuthenticateByUsername(
	db shared.SQLExecutor,
	username string,
	password string,
	deviceInfo DeviceInfo,
) (*string, *string, error) {
	accountFromDB, errFindUser := aw.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return nil, nil, errFindUser
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

func (aw *Service) AuthenticateByEmail(
	db shared.SQLExecutor,
	email string,
	password string,
	deviceInfo DeviceInfo) (*string, *string, error) {
	accountFromDB, errFindUser := aw.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return nil, nil, errFindUser
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

func (aw *Service) Authenticate(
	db shared.SQLExecutor,
	accountFromDB *account.Account,
	password string,
	deviceId string,
	deviceInfo string,
	userAgent string) (*string, *string, error) {

	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return nil, nil, errVerifyPassword
	}
	if !isAuthenticated {
		return nil, nil, shared.Unauthorized
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
		return nil, nil, err
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		aw.config.JWTSecret,
		aw.config.JWTIssuer,
		aw.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return nil, nil, errGenerateAccToken
	}

	return &accessToken, &session.RefreshToken, nil
}

func (aw *Service) RefreshToken(
	db shared.SQLExecutor,
	account *account.Account,
	refreshToken string,
) (*string, *string, error) {

	session, errFind := aw.sessionRepository.FindByHash(refreshToken)
	if errFind != nil {
		return nil, nil, errFind
	}
	if session == nil {
		return nil, nil, SessionNotFound
	}

	valid := session.IsValid()
	if !valid {
		return nil, nil, InvalidSession
	}

	// generate new refresh token
	session.GenerateRefreshToken()
	session.SetLastActiveAt(time.Now().UTC())
	errUpdate := aw.sessionRepository.Update(db, session)
	if errUpdate != nil {
		return nil, nil, errUpdate
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(aw.config.JWTSecret, aw.config.JWTIssuer, aw.config.JWTLifespan, session.GetRandId())
	if errGenerate != nil {
		return nil, nil, errGenerate
	}

	return &newAccessToken, &session.RefreshToken, nil
}

func (aw *Service) Register(
	db shared.SQLExecutor,
	newAccount *account.Account,
	requireVerification bool,
) (*account.Account, *verification.Verification, error) {
	if !requireVerification {
		newAccount.SetEmailVerified()
	}

	errCreateAcc := aw.accountRepository.Create(db, newAccount)

	var newVerification *verification.Verification
	if requireVerification {
		newVerification = verification.NewVerification()
		newVerification.SetAccount(newAccount)
		newVerification.SetCode()
		errCreateVerification := aw.verificationRepository.Create(db, newVerification)
		if errCreateVerification != nil {
			return nil, nil, errCreateVerification
		}
	}
	if errCreateAcc != nil {
		return nil, nil, errCreateAcc
	}

	return newAccount, newVerification, nil
}

type UpdateOpt struct {
	NewName     string
	NewUsername string
	NewAvatar   string
}

func (aw *Service) Update(db shared.SQLExecutor, accountUUID string, opt UpdateOpt) error {
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

func (aw *Service) Verify(db shared.SQLExecutor, accountUUID string, code string) error {
	accountFromDB, errFind := aw.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	verificationFromDB, errFind := aw.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		return errFind
	}

	isValid := verificationFromDB.Validate(code)
	if !isValid {
		return verification.InvalidVerificationCode
	}

	accountFromDB.SetEmailVerified()
	errUpdate := aw.accountRepository.Update(db, accountFromDB)
	if errUpdate != nil {
		return errUpdate
	}

	return aw.verificationRepository.Delete(db, verificationFromDB)
}

func (aw *Service) ResendVerification(db shared.SQLExecutor, accountUUID string) (*verification.Verification, error) {
	accountFromDB, errFind := aw.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	var verificationData *verification.Verification
	verificationData, errFind = aw.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		if errFind == verification.VerificationNotFound {
			verificationData = verification.NewVerification()
			verificationData.SetAccount(accountFromDB)
			verificationData.SetCode()
			errCreateVerification := aw.verificationRepository.Create(db, verificationData)
			if errCreateVerification != nil {
				return nil, errCreateVerification
			}

			return verificationData, nil
		}
	}

	verificationData.SetCode()
	errUpdate := aw.verificationRepository.Update(db, verificationData)
	if errUpdate != nil {
		return nil, errUpdate
	}

	return verificationData, nil
}

func (aw *Service) Delete(db shared.SQLExecutor, account *account.Account) error {
	errDel := aw.accountRepository.Delete(db, account)
	if errDel != nil {
		return errDel
	}

	return nil
}

func (aw *Service) PingSession(db shared.SQLExecutor, sessionRandId string) error {
	session, errFind := aw.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return errFind
	}

	session.SetLastActiveAt(time.Now().UTC())
	errUpdateSess := aw.sessionRepository.Update(db, session)
	if errUpdateSess != nil {
		return errUpdateSess
	}

	return nil
}

func (aw *Service) SeedAccount() error {
	errSeed := aw.accountRepository.SeedAllAccount()
	if errSeed != nil {
		return errSeed
	}

	return nil
}

type AccountFinder struct {
	aw *Service
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

func (aw *Service) Find() *AccountFinder {
	return &AccountFinder{aw: aw}
}

func New(readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Service {
	accountManager := account.NewRepository(readDB, redisClient, app)
	sessionManager := session.NewRepository(readDB, redisClient, app)

	return &Service{
		accountRepository: accountManager,
		sessionRepository: sessionManager,
		config:            app,
	}
}
