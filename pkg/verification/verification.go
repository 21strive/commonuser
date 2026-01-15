package verification

import (
	"context"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
)

type VerificationOps struct {
	accountRepository      *repository.AccountRepository
	sessionRepository      *repository.SessionRepository
	providerRepository     *repository.ProviderRepository
	verificationRepository *repository.VerificationRepository
	config                 *config.App
}

func (v *VerificationOps) Init(
	accountRepository *repository.AccountRepository,
	sessionRepository *repository.SessionRepository,
	providerRepository *repository.ProviderRepository,
	verificationRepository *repository.VerificationRepository,
	config *config.App,
) {
	v.accountRepository = accountRepository
	v.sessionRepository = sessionRepository
	v.providerRepository = providerRepository
	v.verificationRepository = verificationRepository
	v.config = config
}

func (v *VerificationOps) Request(db database.SQLExecutor, accountUUID string) (*model.Verification, error) {
	accountFromDB, errFind := v.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	verificationFromDB, errFind := v.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		if errors.Is(errFind, model.VerificationNotFound) {
			return nil, errFind
		}
	}
	if verificationFromDB != nil {
		return verificationFromDB, nil
	}

	verificationData := model.NewVerification()
	verificationData.SetAccount(accountFromDB)
	verificationData.SetCode()
	errCreateVerification := v.verificationRepository.Create(db, verificationData)
	if errCreateVerification != nil {
		return nil, errCreateVerification
	}

	return verificationData, nil
}

func (v *VerificationOps) Verify(ctx context.Context, db database.SQLExecutor, accountUUID string, code string, sessionId string) (string, error) {
	var newAccessToken string
	accountFromDB, errFind := v.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return newAccessToken, errFind
	}

	verificationFromDB, errFind := v.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		return newAccessToken, errFind
	}

	isValid := verificationFromDB.Validate(code)
	if !isValid {
		return newAccessToken, model.InvalidVerificationCode
	}

	accountFromDB.SetEmailVerified()
	errUpdate := v.accountRepository.Update(ctx, db, accountFromDB)
	if errUpdate != nil {
		return newAccessToken, errUpdate
	}

	errDeleteVerification := v.verificationRepository.Delete(db, verificationFromDB)
	if errDeleteVerification != nil {
		return newAccessToken, errDeleteVerification
	}

	newAccessToken, errGenerateAccToken := accountFromDB.GenerateAccessToken(
		v.config.JWTSecret,
		v.config.JWTIssuer,
		v.config.JWTLifespan,
		sessionId)
	if errGenerateAccToken != nil {
		return newAccessToken, errGenerateAccToken
	}

	return newAccessToken, nil
}

func (v *VerificationOps) Resend(db database.SQLExecutor, accountUUID string) (*model.Verification, error) {
	accountFromDB, errFind := v.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return nil, errFind
	}

	var verificationData *model.Verification
	verificationData, errFind = v.verificationRepository.FindByAccount(accountFromDB)
	if errFind != nil {
		if errFind == model.VerificationNotFound {
			verificationData = model.NewVerification()
			verificationData.SetAccount(accountFromDB)
			verificationData.SetCode()
			errCreateVerification := v.verificationRepository.Create(db, verificationData)
			if errCreateVerification != nil {
				return nil, errCreateVerification
			}

			return verificationData, nil
		}
	}

	verificationData.SetCode()
	errUpdate := v.verificationRepository.Update(db, verificationData)
	if errUpdate != nil {
		return nil, errUpdate
	}

	return verificationData, nil
}

func New() *VerificationOps {
	return &VerificationOps{}
}
