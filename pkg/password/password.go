package password

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/21strive/commonuser/pkg/account"
	"github.com/21strive/commonuser/pkg/session"
	"github.com/redis/go-redis/v9"
	"time"
)

type WithTransaction struct {
	PasswordOps *PasswordOps
	Tx          *sql.Tx
}

func (w *WithTransaction) RequestResetPassword(ctx context.Context, account *model.Account, expiration *time.Time) (*model.ResetPassword, error) {
	return w.PasswordOps.requestResetPassword(ctx, w.Tx, account, expiration)
}

func (w *WithTransaction) ValidateResetPassword(ctx context.Context, pipe redis.Pipeliner, account *model.Account, newPassword string, token string) error {
	return w.PasswordOps.validateResetPassword(ctx, pipe, w.Tx, account, newPassword, token)
}

func (w *WithTransaction) DeleteResetPasswordRequest(ctx context.Context, accountUUID string) error {
	return w.PasswordOps.deleteResetPasswordRequest(ctx, w.Tx, accountUUID)
}

func (w *WithTransaction) UpdateResetPasswordRequest(ctx context.Context, pipe redis.Pipeliner, accountUUID string, oldPassword string, newPassword string) error {
	return w.PasswordOps.updateResetPasswordRequest(ctx, pipe, w.Tx, accountUUID, oldPassword, newPassword)
}

type PasswordOps struct {
	writeDB                 *sql.DB
	resetPasswordRepository *repository.ResetPasswordRepository
	sessionOps              *session.SessionOps
	accountOps              *account.AccountOps
}

func (pu *PasswordOps) SetWriteDB(db *sql.DB) {
	pu.writeDB = db
}

func (pu *PasswordOps) requestResetPassword(ctx context.Context, db types.SQLExecutor, account *model.Account, expiration *time.Time) (*model.ResetPassword, error) {
	ticketFromDB, errFind := pu.resetPasswordRepository.FindRequest(account)
	if errFind != nil {
		if !errors.Is(errFind, model.ResetPasswordTicketNotFound) {
			return nil, errFind
		}
	}
	if ticketFromDB != nil {
		if ticketFromDB.IsExpired() {
			errDeleteTicket := pu.resetPasswordRepository.DeleteAllRequests(ctx, db, account)
			if errDeleteTicket != nil {
				return nil, errDeleteTicket
			}
		} else {
			return ticketFromDB, nil
		}
	}

	newResetPasswordTicket := model.NewResetPasswordRequest()
	newResetPasswordTicket.SetAccount(account)
	newResetPasswordTicket.SetToken()
	if expiration != nil {
		newResetPasswordTicket.SetExpiredAt(expiration)
	}
	errCreate := pu.resetPasswordRepository.CreateRequest(ctx, db, newResetPasswordTicket)
	if errCreate != nil {
		return nil, errCreate
	}

	return newResetPasswordTicket, nil
}

func (pu *PasswordOps) RequestResetPassword(ctx context.Context, account *model.Account, expiration *time.Time) (*model.ResetPassword, error) {
	return pu.requestResetPassword(ctx, pu.writeDB, account, expiration)
}

func (pu *PasswordOps) validateResetPassword(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account, newPassword string, token string) error {
	ticketFromDB, errFind := pu.resetPasswordRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := ticketFromDB.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, model.ResetPasswordRequestExpired) {
			pu.resetPasswordRepository.DeleteAllRequests(ctx, db, account)
			return errValidate
		}
		return errValidate
	}

	account.SetPassword(newPassword)
	account.SetUpdatedAt(time.Now().UTC())
	var errUpdateAccount error
	if pipe != nil {
		errUpdateAccount = pu.accountOps.WithTransaction(pipe, db.(*sql.Tx)).Update(ctx, account)
	} else {
		errUpdateAccount = pu.accountOps.Update(ctx, account)
	}
	if errUpdateAccount != nil {
		return errUpdateAccount
	}

	errUpdateTicket := pu.resetPasswordRepository.DeleteAllRequests(ctx, db, account)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	var errRevoke error
	if pipe != nil {
		errRevoke = pu.sessionOps.Revoke
	}
	errRevoke := pu.sessionRepository.RevokeAll(ctx, db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (pu *PasswordOps) ValidateResetPassword(ctx context.Context, account *model.Account, newPassword string, token string) error {
	return pu.validateResetPassword(ctx, nil, pu.writeDB, account, newPassword, token)
}

func (pu *PasswordOps) deleteResetPasswordRequest(ctx context.Context, db types.SQLExecutor, accountUUID string) error {
	accountFromDB, errFind := pu.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	return pu.resetPasswordRepository.DeleteAllRequests(ctx, db, accountFromDB)
}

func (pu *PasswordOps) DeleteResetPasswordRequest(ctx context.Context, accountUUID string) error {
	return pu.deleteResetPasswordRequest(ctx, pu.writeDB, accountUUID)
}

func (pu *PasswordOps) updateResetPasswordRequest(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountUUID string, oldPassword string, newPassword string) error {
	accountFromDB, errFind := pu.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	isValid, errValidate := accountFromDB.VerifyPassword(oldPassword)
	if errValidate != nil {
		return errValidate
	}
	if !isValid {
		return model.Unauthorized
	}

	accountFromDB.SetPassword(newPassword)

	// revoke all running sessions
	errRevoke := pu.sessionRepository.RevokeAll(ctx, db, accountFromDB)
	if errRevoke != nil {
		return errRevoke
	}

	return pu.accountRepository.Update(ctx, pipe, db, accountFromDB)
}

func (pu *PasswordOps) UpdateResetPasswordRequest(ctx context.Context, accountUUID string, oldPassword string, newPassword string) error {
	return pu.updateResetPasswordRequest(ctx, nil, pu.writeDB, accountUUID, oldPassword, newPassword)
}

func New(resetPasswordRepository *repository.ResetPasswordRepository, sessionOps *session.SessionOps, accountOps *account.AccountOps) *PasswordOps {
	return &PasswordOps{
		resetPasswordRepository: resetPasswordRepository,
		sessionOps:              sessionOps,
		accountOps:              accountOps,
	}
}
