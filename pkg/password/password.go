package password

import (
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"time"
)

type PasswordOps struct {
	resetPasswordRepository repository.ResetPasswordRepository
}

func (pu *PasswordOps) RequestReset(db database.SQLExecutor, account *model.Account, expiration *time.Time) (*model.ResetPassword, error) {
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

	newResetPasswordTicket := model.New()
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

func (pu *PasswordOps) ValidateReset(db database.SQLExecutor, account *model.Account, newPassword string, token string) error {
	ticketFromDB, errFind := pu.s.resetPasswordRepository.Find(account)
	if errFind != nil {
		return errFind
	}

	errValidate := ticketFromDB.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, constant.RequestExpired) {
			pu.s.resetPasswordRepository.DeleteAll(db, account)
			return constant.RequestExpired
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

func (pu *PasswordOps) DeleteReset(db database.SQLExecutor, accountUUID string) error {
	accountFromDB, errFind := pu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	return pu.s.resetPasswordRepository.DeleteAll(db, accountFromDB)
}

func (pu *PasswordOps) Update(db database.SQLExecutor, accountUUID string, oldPassword string, newPassword string) error {
	accountFromDB, errFind := pu.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	isValid, errValidate := accountFromDB.VerifyPassword(oldPassword)
	if errValidate != nil {
		return errValidate
	}
	if !isValid {
		return constant.Unauthorized
	}

	accountFromDB.SetPassword(newPassword)

	// revoke all running sessions
	errRevoke := pu.s.sessionRepository.RevokeAll(db, accountFromDB)
	if errRevoke != nil {
		return errRevoke
	}

	return pu.s.accountRepository.Update(db, accountFromDB)
}

func (aw *Service) Password() *PasswordOps {
	return &PasswordOps{s: aw}
}
