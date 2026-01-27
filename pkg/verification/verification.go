package verification

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/21strive/commonuser/pkg/account"
	"github.com/redis/go-redis/v9"
)

type WithTransaction struct {
	VerificationOps *VerificationOps
	Tx              *sql.Tx
}

func (w *WithTransaction) Request(ctx context.Context, newAccount *model.Account) (*model.Verification, error) {
	return w.VerificationOps.request(ctx, w.Tx, newAccount)
}

func (w *WithTransaction) Verify(ctx context.Context, pipe redis.Pipeliner, newAccount *model.Account, code string, sessionId string) (string, error) {
	return w.VerificationOps.verify(ctx, pipe, w.Tx, newAccount, code, sessionId)
}

func (w *WithTransaction) Resend(ctx context.Context, newAccount *model.Account) (*model.Verification, error) {
	return w.VerificationOps.resend(ctx, w.Tx, newAccount)
}

type VerificationOps struct {
	writeDB                *sql.DB
	verificationRepository *repository.VerificationRepository
	accountOps             *account.AccountOps
	config                 *config.App
}

func (v *VerificationOps) SetWriteDB(db *sql.DB) {
	v.writeDB = db
}

func (v *VerificationOps) WithTransaction(tx *sql.Tx) *WithTransaction {
	return &WithTransaction{VerificationOps: v, Tx: tx}
}

func (v *VerificationOps) request(ctx context.Context, db types.SQLExecutor, account *model.Account) (*model.Verification, error) {
	verificationFromDB, errFind := v.verificationRepository.FindByAccount(account)
	if errFind != nil {
		if errors.Is(errFind, model.VerificationNotFound) {
			return nil, errFind
		}
	}
	if verificationFromDB != nil {
		return verificationFromDB, nil
	}

	verificationData := model.NewVerification()
	verificationData.SetAccount(account)
	verificationData.SetCode()
	errCreateVerification := v.verificationRepository.Create(ctx, db, verificationData)
	if errCreateVerification != nil {
		return nil, errCreateVerification
	}

	return verificationData, nil
}

func (v *VerificationOps) Request(ctx context.Context, account *model.Account) (*model.Verification, error) {
	return v.request(ctx, v.writeDB, account)
}

func (v *VerificationOps) verify(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, newAccount *model.Account, code string, sessionId string) (string, error) {
	var newAccessToken string

	verificationFromDB, errFind := v.verificationRepository.FindByAccount(newAccount)
	if errFind != nil {
		return newAccessToken, errFind
	}

	isValid := verificationFromDB.Validate(code)
	if !isValid {
		return newAccessToken, model.InvalidVerificationCode
	}

	newAccount.SetEmailVerified()
	var errUpdateAcc error
	if pipe != nil {
		errUpdateAcc = v.accountOps.WithTransaction(pipe, db.(*sql.Tx)).Update(ctx, newAccount)

	} else {
		errUpdateAcc = v.accountOps.Update(ctx, newAccount)
	}
	if errUpdateAcc != nil {
		return newAccessToken, errUpdateAcc
	}

	errDeleteVerification := v.verificationRepository.Delete(ctx, db, verificationFromDB)
	if errDeleteVerification != nil {
		return newAccessToken, errDeleteVerification
	}

	newAccessToken, errGenerateAccToken := newAccount.GenerateAccessToken(
		v.config.JWTSecret,
		v.config.JWTIssuer,
		v.config.JWTLifespan,
		sessionId)
	if errGenerateAccToken != nil {
		return newAccessToken, errGenerateAccToken
	}

	return newAccessToken, nil
}

func (v *VerificationOps) Verify(ctx context.Context, newAccount *model.Account, code string, sessionId string) (string, error) {
	return v.verify(ctx, nil, v.writeDB, newAccount, code, sessionId)
}

func (v *VerificationOps) resend(ctx context.Context, db types.SQLExecutor, newAccount *model.Account) (*model.Verification, error) {
	verificationData, errFind := v.verificationRepository.FindByAccount(newAccount)
	if errFind != nil {
		if errors.Is(errFind, model.VerificationNotFound) {
			verificationData = model.NewVerification()
			verificationData.SetAccount(newAccount)
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

func (v *VerificationOps) Resend(ctx context.Context, newAccount *model.Account) (*model.Verification, error) {
	return v.resend(ctx, v.writeDB, newAccount)
}

func New(verificationRepository *repository.VerificationRepository, accountOps *account.AccountOps, config *config.App) *VerificationOps {
	return &VerificationOps{
		verificationRepository: verificationRepository,
		accountOps:             accountOps,
		config:                 config,
	}
}
