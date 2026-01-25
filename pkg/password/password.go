package password

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
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
	sessionRepository       *repository.SessionRepository
	accountRepository       *repository.AccountRepository
}

func (pu *PasswordOps) Init(
	resetPasswordRepository *repository.ResetPasswordRepository,
	sessionRepository *repository.SessionRepository,
	accountRepository *repository.AccountRepository,
) {
	pu.resetPasswordRepository = resetPasswordRepository
	pu.sessionRepository = sessionRepository
	pu.accountRepository = accountRepository
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
	errUpdate := pu.accountRepository.Update(ctx, pipe, db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errUpdateTicket := pu.resetPasswordRepository.DeleteAllRequests(ctx, db, account)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
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

func New() *PasswordOps {
	return &PasswordOps{}
}
