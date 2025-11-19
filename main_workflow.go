package commonuser

import (
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/provider"
	"github.com/21strive/commonuser/reset_password"
	"github.com/21strive/commonuser/session"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/commonuser/update_email"
	"github.com/21strive/commonuser/verification"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

var SessionNotFound = errors.New("session not found!")
var InvalidSession = errors.New("invalid session!")

type Service struct {
	accountRepository       *account.Repository
	sessionRepository       *session.Repository
	verificationRepository  *verification.Repository
	updateEmailRepository   *update_email.Repository
	resetPasswordRepository *reset_password.Repository
	providerRepository      *provider.Repository
	config                  *config.App
}

func (aw *Service) AccountBase() *redifu.Base[*account.Account] {
	return aw.accountRepository.GetBase()
}

func (aw *Service) SessionBase() *redifu.Base[*session.Session] {
	return aw.sessionRepository.GetBase()
}

func (aw *Service) Config() *config.App {
	return aw.config
}

type DeviceInfo struct {
	DeviceId   string `json:"deviceId"`
	DeviceType string `json:"deviceType"`
	UserAgent  string `json:"userAgent"`
}

type Authenticate struct {
	s *Service
}

func (au *Authenticate) ByProvider(db shared.SQLExecutor, issuer string, sub string, deviceInfo DeviceInfo) (string, string, error) {
	providerFromDB, errFind := au.s.providerRepository.Find(sub, issuer)
	if errFind != nil {
		return "", "", errFind
	}

	accountFromDB, errFind := au.s.accountRepository.FindByUUID(providerFromDB.AccountUUID)
	if errFind != nil {
		return "", "", errFind
	}

	return au.GenerateToken(db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authenticate) ByUsername(db shared.SQLExecutor, username string, password string, deviceInfo DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.s.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.AuthenticatePassword(db, accountFromDB, password, deviceInfo)
}

func (au *Authenticate) ByEmail(db shared.SQLExecutor, email string, password string, deviceInfo DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := au.s.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return au.AuthenticatePassword(db, accountFromDB, password, deviceInfo)
}

func (au *Authenticate) AuthenticatePassword(db shared.SQLExecutor, accountFromDB *account.Account, password string, deviceInfo DeviceInfo) (string, string, error) {
	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return "", "", errVerifyPassword
	}
	if !isAuthenticated {
		return "", "", shared.Unauthorized
	}

	return au.GenerateToken(db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (au *Authenticate) GenerateToken(db shared.SQLExecutor, accountFromDB *account.Account, deviceId string, deviceType string, userAgent string) (string, string, error) {
	session := session.NewSession()

	session.SetDeviceId(deviceId)
	session.SetDeviceType(deviceType)
	session.SetUserAgent(userAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(au.s.config.TokenLifespan)
	session.GenerateRefreshToken()

	err := au.s.sessionRepository.Create(db, session)
	if err != nil {
		return "", "", err
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		au.s.config.JWTSecret,
		au.s.config.JWTIssuer,
		au.s.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return "", "", errGenerateAccToken
	}

	return accessToken, session.RefreshToken, nil
}

func (aw *Service) Authenticate() *Authenticate {
	return &Authenticate{s: aw}
}

func (aw *Service) RegisterWithProvider(db shared.SQLExecutor, newAccount *account.Account, newProvider *provider.Provider) error {
	errCreateProvider := aw.providerRepository.Create(db, newProvider)
	if errCreateProvider != nil {
		return errCreateProvider
	}

	_, errRegister := aw.Register(db, newAccount, false)
	return errRegister
}

func (aw *Service) Register(db shared.SQLExecutor, newAccount *account.Account, requireVerification bool) (*string, error) {
	if !requireVerification {
		newAccount.SetEmailVerified()
	}

	errCreateAcc := aw.accountRepository.Create(db, newAccount)
	if errCreateAcc != nil {
		return nil, errCreateAcc
	}

	var verificationCode string
	var newVerification *verification.Verification
	if requireVerification {
		newVerification = verification.New()
		newVerification.SetAccount(newAccount)
		verificationCode = newVerification.SetCode()
		errCreateVerification := aw.verificationRepository.Create(db, newVerification)
		if errCreateVerification != nil {
			return nil, errCreateVerification
		}
	}

	return &verificationCode, nil
}

type UpdateOpt struct {
	NewName     string
	NewUsername string
	NewAvatar   string
}

func (aw *Service) Update(db shared.SQLExecutor, accountUUID string, opt UpdateOpt) (*account.Account, error) {
	accountFromDB, errFind := aw.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
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
		return nil, errSet
	}

	errUpdateRef := aw.accountRepository.UpdateReference(accountFromDB, oldUsername, accountFromDB.Username)
	if errUpdateRef != nil {
		return nil, errUpdateRef
	}

	return accountFromDB, nil
}

type Verification struct {
	s *Service
}

func (v *Verification) Request(db shared.SQLExecutor, accountUUID string) (*verification.Verification, error) {
	accountFromDB, errFind := v.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	verificationFromDB, errFind := v.s.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		if errors.Is(errFind, verification.VerificationNotFound) {
			return nil, errFind
		}
	}
	if verificationFromDB != nil {
		return verificationFromDB, nil
	}

	verificationData := verification.New()
	verificationData.SetAccount(accountFromDB)
	verificationData.SetCode()
	errCreateVerification := v.s.verificationRepository.Create(db, verificationData)
	if errCreateVerification != nil {
		return nil, errCreateVerification
	}

	return verificationData, nil
}

func (v *Verification) Verify(db shared.SQLExecutor, accountUUID string, code string, sessionId string) (string, error) {
	var newAccessToken string
	accountFromDB, errFind := v.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return newAccessToken, errFind
	}

	verificationFromDB, errFind := v.s.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		return newAccessToken, errFind
	}

	isValid := verificationFromDB.Validate(code)
	if !isValid {
		return newAccessToken, verification.InvalidVerificationCode
	}

	accountFromDB.SetEmailVerified()
	errUpdate := v.s.accountRepository.Update(db, accountFromDB)
	if errUpdate != nil {
		return newAccessToken, errUpdate
	}

	errDeleteVerification := v.s.verificationRepository.Delete(db, verificationFromDB)
	if errDeleteVerification != nil {
		return newAccessToken, errDeleteVerification
	}

	newAccessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		v.s.config.JWTSecret,
		v.s.config.JWTIssuer,
		v.s.config.JWTLifespan,
		sessionId)
	if errGenerateAccToken != nil {
		return newAccessToken, errGenerateAccToken
	}

	return newAccessToken, nil
}

func (v *Verification) Resend(db shared.SQLExecutor, accountUUID string) (*verification.Verification, error) {
	accountFromDB, errFind := v.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	var verificationData *verification.Verification
	verificationData, errFind = v.s.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		if errFind == verification.VerificationNotFound {
			verificationData = verification.New()
			verificationData.SetAccount(accountFromDB)
			verificationData.SetCode()
			errCreateVerification := v.s.verificationRepository.Create(db, verificationData)
			if errCreateVerification != nil {
				return nil, errCreateVerification
			}

			return verificationData, nil
		}
	}

	verificationData.SetCode()
	errUpdate := v.s.verificationRepository.Update(db, verificationData)
	if errUpdate != nil {
		return nil, errUpdate
	}

	return verificationData, nil
}

func (aw *Service) Verification() *Verification {
	return &Verification{s: aw}
}

func (aw *Service) Delete(db shared.SQLExecutor, account *account.Account) error {
	errDel := aw.accountRepository.Delete(db, account)
	if errDel != nil {
		return errDel
	}

	return nil
}

type Session struct {
	s *Service
}

func (s *Session) Create(db shared.SQLExecutor, session *session.Session) error {
	return s.s.sessionRepository.Create(db, session)
}

func (s *Session) Ping(db shared.SQLExecutor, sessionRandId string) error {
	sessionFromDB, errFind := s.s.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return errFind
	}

	if sessionFromDB.IsValid() {
		return session.InvalidSession
	}

	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	return s.s.sessionRepository.Update(db, sessionFromDB)
}

func (s *Session) Revoke(db shared.SQLExecutor, sessionRandId string) error {
	session, errFind := s.s.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return errFind
	}

	session.SetUpdatedAt(time.Now().UTC())
	session.Revoke()
	return s.s.sessionRepository.Update(db, session)
}

func (s *Session) Refresh(db shared.SQLExecutor, account *account.Account, sessionRandId string) (string, string, error) {
	sessionFromDB, errFind := s.s.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return "", "", errFind
	}
	if !sessionFromDB.IsValid() {
		return "", "", session.InvalidSession
	}

	sessionFromDB.SetUpdatedAt(time.Now().UTC())
	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	sessionFromDB.SetLifeSpan(s.s.config.TokenLifespan)
	sessionFromDB.GenerateRefreshToken()
	errUpdate := s.s.sessionRepository.Update(db, sessionFromDB)
	if errUpdate != nil {
		return "", "", errUpdate
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(s.s.config.JWTSecret, s.s.config.JWTIssuer, s.s.config.JWTLifespan, sessionFromDB.GetRandId())
	if errGenerate != nil {
		return "", "", errGenerate
	}

	return newAccessToken, sessionFromDB.RefreshToken, nil
}

func (s *Session) SeedByAccount(accountUUID string) error {
	return s.s.sessionRepository.SeedByAccount(accountUUID)
}

func (s *Session) PurgeInvalid(db shared.SQLExecutor) error {
	return s.s.sessionRepository.PurgeInvalid(db)
}

func (aw *Service) Session() *Session {
	return &Session{s: aw}
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

type Email struct {
	s *Service
}

func (eu *Email) RequestUpdate(
	db shared.SQLExecutor,
	account *account.Account,
	newEmailAddress string,
) (*update_email.UpdateEmail, error) {

	requestFromDB, errFind := eu.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		if !errors.Is(errFind, update_email.TicketNotFound) {
			return nil, errFind
		}
	}
	if requestFromDB != nil {
		if requestFromDB.IsExpired() {
			errDeleteRequest := eu.s.updateEmailRepository.DeleteAll(db, account)
			if errDeleteRequest != nil {
				return nil, errDeleteRequest
			}
		} else {
			return requestFromDB, nil
		}
	}

	updateEmailRequest := update_email.New()
	updateEmailRequest.SetAccount(account)
	updateEmailRequest.SetPreviousEmailAddress(account.Base.Email)
	updateEmailRequest.SetNewEmailAddress(newEmailAddress)
	updateEmailRequest.SetExpiration()
	_, errGen := updateEmailRequest.SetToken()
	if errGen != nil {
		return nil, errGen
	}
	_, errGen = updateEmailRequest.SetRevokeToken()
	if errGen != nil {
		return nil, errGen
	}

	errCreateTicket := eu.s.updateEmailRepository.CreateRequest(db, updateEmailRequest)
	if errCreateTicket != nil {
		return nil, errCreateTicket
	}

	return updateEmailRequest, nil
}

func (eu *Email) ValidateUpdate(
	db shared.SQLExecutor,
	accountUUID string,
	token string,
) error {
	account, errFind := eu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := eu.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}
	if request.Processed {
		return nil
	}

	errValidate := request.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, shared.RequestExpired) {
			eu.s.updateEmailRepository.DeleteAll(db, account)
			return shared.RequestExpired
		}
		return errValidate
	}

	account.SetEmail(request.NewEmailAddress)
	errUpdate := eu.s.accountRepository.Update(db, account)
	if errUpdate != nil {
		return errUpdate
	}

	request.SetProcessed()
	errUpdateTicket := eu.s.updateEmailRepository.UpdateRequest(db, request)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	errRevoke := eu.s.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (eu *Email) RevokeUpdate(
	db shared.SQLExecutor,
	accountUUID string,
	revokeToken string,
) error {
	account, errFind := eu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := eu.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := request.ValidateRevoke(revokeToken)
	if errValidate != nil {
		return errValidate
	}

	account.SetEmail(request.PreviousEmailAddress)
	errUpdate := eu.s.accountRepository.Update(db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errDeleteTicket := eu.s.updateEmailRepository.DeleteAll(db, account)
	if errDeleteTicket != nil {
		return errDeleteTicket
	}

	// revoke all running sessions
	errRevoke := eu.s.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (eu *Email) DeleteUpdateRequest(db shared.SQLExecutor, account *account.Account) error {
	return eu.s.updateEmailRepository.DeleteAll(db, account)
}

func (aw *Service) EmailUpdate() *Email {
	return &Email{s: aw}
}

type Password struct {
	s *Service
}

func (pu *Password) RequestReset(db shared.SQLExecutor, account *account.Account, expiration *time.Time) (*reset_password.ResetPassword, error) {
	ticketFromDB, errFind := pu.s.resetPasswordRepository.Find(account)
	if errFind != nil {
		if !errors.Is(errFind, reset_password.TicketNotFound) {
			return nil, errFind
		}
	}
	if ticketFromDB != nil {
		if ticketFromDB.IsExpired() {
			errDeleteTicket := pu.s.resetPasswordRepository.DeleteAll(db, account)
			if errDeleteTicket != nil {
				return nil, errDeleteTicket
			}
		} else {
			return ticketFromDB, nil
		}
	}

	newResetPasswordTicket := reset_password.New()
	newResetPasswordTicket.SetAccount(account)
	newResetPasswordTicket.SetToken()
	if expiration != nil {
		newResetPasswordTicket.SetExpiredAt(expiration)
	}
	errCreate := pu.s.resetPasswordRepository.Create(db, newResetPasswordTicket)
	if errCreate != nil {
		return nil, errCreate
	}

	return newResetPasswordTicket, nil
}

func (pu *Password) ValidateReset(db shared.SQLExecutor, account *account.Account, newPassword string, token string) error {
	ticketFromDB, errFind := pu.s.resetPasswordRepository.Find(account)
	if errFind != nil {
		return errFind
	}

	errValidate := ticketFromDB.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, shared.RequestExpired) {
			pu.s.resetPasswordRepository.DeleteAll(db, account)
			return shared.RequestExpired
		}
		return errValidate
	}

	account.SetPassword(newPassword)
	account.SetUpdatedAt(time.Now().UTC())
	errUpdate := pu.s.accountRepository.Update(db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errUpdateTicket := pu.s.resetPasswordRepository.DeleteAll(db, account)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	errRevoke := pu.s.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (pu *Password) DeleteReset(db shared.SQLExecutor, accountUUID string) error {
	accountFromDB, errFind := pu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	return pu.s.resetPasswordRepository.DeleteAll(db, accountFromDB)
}

func (pu *Password) Update(db shared.SQLExecutor, accountUUID string, oldPassword string, newPassword string) error {
	accountFromDB, errFind := pu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	isValid, errValidate := accountFromDB.VerifyPassword(oldPassword)
	if errValidate != nil {
		return errValidate
	}
	if !isValid {
		return shared.Unauthorized
	}

	accountFromDB.SetPassword(newPassword)

	// revoke all running sessions
	errRevoke := pu.s.sessionRepository.RevokeAll(db, accountFromDB)
	if errRevoke != nil {
		return errRevoke
	}

	return pu.s.accountRepository.Update(db, accountFromDB)
}

func (aw *Service) Password() *Password {
	return &Password{s: aw}
}

func New(readDB *sql.DB, redisClient redis.UniversalClient, app *config.App) *Service {
	accountManager := account.NewRepository(readDB, redisClient, app)
	sessionManager := session.NewRepository(readDB, redisClient, app)
	verificationManager := verification.NewRepository(readDB, app)
	updateEmailManager := update_email.NewUpdateEmailManager(readDB, app)
	resetPasswordManager := reset_password.NewRepository(readDB, app)
	providerRepository := provider.NewRepository(readDB, app)

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
