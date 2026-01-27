package verification

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/redis/go-redis/v9"
)

type WithTransaction struct {
	VerificationOps *VerificationOps
	Tx              *sql.Tx
}

func (w *WithTransaction) Request(ctx context.Context, accountUUID string) (*model.Verification, error) {
	return w.VerificationOps.request(ctx, w.Tx, accountUUID)
}

func (w *WithTransaction) Verify(ctx context.Context, pipe redis.Pipeliner, accountUUID string, code string, sessionId string) (string, error) {
	return w.VerificationOps.verify(ctx, pipe, w.Tx, accountUUID, code, sessionId)
}

func (w *WithTransaction) Resend(ctx context.Context, accountUUID string) (*model.Verification, error) {
	return w.VerificationOps.resend(ctx, w.Tx, accountUUID)
}

type VerificationOps struct {
	writeDB                *sql.DB
	accountRepository      *repository.AccountRepository
	sessionRepository      *repository.SessionRepository
	providerRepository     *repository.ProviderRepository
	verificationRepository *repository.VerificationRepository
	config                 *config.App
}

func (v *VerificationOps) WithTransaction(tx *sql.Tx) *WithTransaction {
	return &WithTransaction{VerificationOps: v, Tx: tx}
}

func (v *VerificationOps) request(ctx context.Context, db types.SQLExecutor, accountUUID string) (*model.Verification, error) {
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
	errCreateVerification := v.verificationRepository.Create(ctx, db, verificationData)
	if errCreateVerification != nil {
		return nil, errCreateVerification
	}

	return verificationData, nil
}

func (v *VerificationOps) Request(ctx context.Context, accountUUID string) (*model.Verification, error) {
	return v.request(ctx, v.writeDB, accountUUID)
}

func (v *VerificationOps) verify(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountUUID string, code string, sessionId string) (string, error) {
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
	errUpdate := v.accountRepository.Update(ctx, pipe, db, accountFromDB)
	if errUpdate != nil {
		return newAccessToken, errUpdate
	}

	errDeleteVerification := v.verificationRepository.Delete(ctx, db, verificationFromDB)
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

func (v *VerificationOps) Verify(ctx context.Context, accountUUID string, code string, sessionId string) (string, error) {
	return v.verify(ctx, nil, v.writeDB, accountUUID, code, sessionId)
}

func (v *VerificationOps) resend(ctx context.Context, db types.SQLExecutor, accountUUID string) (*model.Verification, error) {
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
			errCreateVerification := v.verificationRepository.Create(ctx, db, verificationData)
			if errCreateVerification != nil {
				return nil, errCreateVerification
			}

			return verificationData, nil
		}
	}

	verificationData.SetCode()
	errUpdate := v.verificationRepository.Update(ctx, db, verificationData)
	if errUpdate != nil {
		return nil, errUpdate
	}

	return verificationData, nil
}

func (v *VerificationOps) Resend(ctx context.Context, accountUUID string) (*model.Verification, error) {
	return v.resend(ctx, v.writeDB, accountUUID)
}

func New(repositoryPool *database.RepositoryPool, config *config.App, writeDB *sql.DB) *VerificationOps {
	return &VerificationOps{
		writeDB:                writeDB,
		accountRepository:      repositoryPool.AccountRepository,
		sessionRepository:      repositoryPool.SessionRepository,
		providerRepository:     repositoryPool.ProviderRepository,
		verificationRepository: repositoryPool.VerificationRepository,
		config:                 config,
	}
}
