package password

import (
	"context"
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"time"
)

type PasswordOps struct {
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

func (pu *PasswordOps) RequestResetPassword(db database.SQLExecutor, account *model.Account, expiration *time.Time) (*model.ResetPassword, error) {
	ticketFromDB, errFind := pu.resetPasswordRepository.FindRequest(account)
	if errFind != nil {
		if !errors.Is(errFind, model.ResetPasswordTicketNotFound) {
			return nil, errFind
		}
	}
	if ticketFromDB != nil {
		if ticketFromDB.IsExpired() {
			errDeleteTicket := pu.resetPasswordRepository.DeleteAllRequests(db, account)
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
	errCreate := pu.resetPasswordRepository.CreateRequest(db, newResetPasswordTicket)
	if errCreate != nil {
		return nil, errCreate
	}

	return newResetPasswordTicket, nil
}

func (pu *PasswordOps) ValidateResetPassword(ctx context.Context, db database.SQLExecutor, account *model.Account, newPassword string, token string) error {
	ticketFromDB, errFind := pu.resetPasswordRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := ticketFromDB.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, model.ResetPasswordRequestExpired) {
			pu.resetPasswordRepository.DeleteAllRequests(db, account)
			return errValidate
		}
		return errValidate
	}

	account.SetPassword(newPassword)
	account.SetUpdatedAt(time.Now().UTC())
	errUpdate := pu.accountRepository.Update(ctx, db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errUpdateTicket := pu.resetPasswordRepository.DeleteAllRequests(db, account)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	errRevoke := pu.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (pu *PasswordOps) DeleteResetPasswordRequest(db database.SQLExecutor, accountUUID string) error {
	accountFromDB, errFind := pu.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	return pu.resetPasswordRepository.DeleteAllRequests(db, accountFromDB)
}

func (pu *PasswordOps) UpdateResetPasswordRequest(ctx context.Context, db database.SQLExecutor, accountUUID string, oldPassword string, newPassword string) error {
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
	errRevoke := pu.sessionRepository.RevokeAll(db, accountFromDB)
	if errRevoke != nil {
		return errRevoke
	}

	return pu.accountRepository.Update(ctx, db, accountFromDB)
}

func New() *PasswordOps {
	return &PasswordOps{}
}
