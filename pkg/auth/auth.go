package auth

import (
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"time"
)

type AuthOps struct {
	accountRepository  *repository.AccountRepository
	sessionRepository  *repository.AccountRepository
	providerRepository *repository.AccountRepository
	config             *config.App
}

func (o *AuthOps) ByProvider(db database.SQLExecutor, issuer string, sub string, deviceInfo DeviceInfo) (string, string, error) {
	providerFromDB, errFind := o.providerRepository.Find(sub, issuer)
	if errFind != nil {
		return "", "", errFind
	}

	accountFromDB, errFind := o.accountRepository.FindByUUID(providerFromDB.AccountUUID)
	if errFind != nil {
		return "", "", errFind
	}

	return o.GenerateToken(db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (o *AuthOps) ByUsername(db database.SQLExecutor, username string, password string, deviceInfo DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := o.accountRepository.FindByUsername(username)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return o.AuthenticatePassword(db, accountFromDB, password, deviceInfo)
}

func (o *AuthOps) ByEmail(db database.SQLExecutor, email string, password string, deviceInfo DeviceInfo) (string, string, error) {
	accountFromDB, errFindUser := o.accountRepository.FindByEmail(email)
	if errFindUser != nil {
		return "", "", errFindUser
	}

	return o.AuthenticatePassword(db, accountFromDB, password, deviceInfo)
}

func (o *AuthOps) AuthenticatePassword(db database.SQLExecutor, accountFromDB *model.Account, password string, deviceInfo DeviceInfo) (string, string, error) {
	isAuthenticated, errVerifyPassword := accountFromDB.VerifyPassword(password)
	if errVerifyPassword != nil {
		return "", "", errVerifyPassword
	}
	if !isAuthenticated {
		return "", "", constant.Unauthorized
	}

	return o.GenerateToken(db, accountFromDB, deviceInfo.DeviceId, deviceInfo.DeviceType, deviceInfo.UserAgent)
}

func (o *AuthOps) GenerateToken(db database.SQLExecutor, accountFromDB *model.Account, deviceId string, deviceType string, userAgent string) (string, string, error) {
	session := model.NewSession()
	session.SetDeviceId(deviceId)
	session.SetDeviceType(deviceType)
	session.SetUserAgent(userAgent)
	session.SetAccountUUID(accountFromDB.GetUUID())
	session.SetLastActiveAt(time.Now().UTC())
	session.SetLifeSpan(o.config.TokenLifespan)
	session.GenerateRefreshToken()

	err := o.sessionRepository.Create(db, session)
	if err != nil {
		return "", "", err
	}

	accessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		o.config.JWTSecret,
		o.config.JWTIssuer,
		o.config.JWTLifespan,
		session.GetRandId())
	if errGenerateAccToken != nil {
		return "", "", errGenerateAccToken
	}

	return accessToken, session.RefreshToken, nil
}
