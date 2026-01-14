package email

import (
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
)

type EmailOps struct {
	updateEmailRepository repository.UpdateEmailRepository
}

func (e *EmailOps) RequestUpdate(
	db database.SQLExecutor,
	account *model.Account,
	newEmailAddress string,
) (*model.UpdateEmail, error) {

	requestFromDB, errFind := e.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		if !errors.Is(errFind, update_email.TicketNotFound) {
			return nil, errFind
		}
	}
	if requestFromDB != nil {
		if requestFromDB.IsExpired() {
			errDeleteRequest := e.s.updateEmailRepository.DeleteAll(db, account)
			if errDeleteRequest != nil {
				return nil, errDeleteRequest
			}
		} else {
			return requestFromDB, nil
		}
	}

	updateEmailRequest := model.NewAccount()
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

	errCreateTicket := e.s.updateEmailRepository.CreateRequest(db, updateEmailRequest)
	if errCreateTicket != nil {
		return nil, errCreateTicket
	}

	return updateEmailRequest, nil
}

func (e *EmailOps) ValidateUpdate(
	db database.SQLExecutor,
	accountUUID string,
	token string,
) error {
	account, errFind := e.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := e.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}
	if request.Processed {
		return nil
	}

	errValidate := request.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, constant.RequestExpired) {
			e.s.updateEmailRepository.DeleteAll(db, account)
			return constant.RequestExpired
		}
		return errValidate
	}

	account.SetEmail(request.NewEmailAddress)
	errUpdate := e.s.accountRepository.Update(db, account)
	if errUpdate != nil {
		return errUpdate
	}

	request.SetProcessed()
	errUpdateTicket := e.s.updateEmailRepository.UpdateRequest(db, request)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	errRevoke := e.s.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (e *EmailOps) RevokeUpdate(
	db database.SQLExecutor,
	accountUUID string,
	revokeToken string,
) error {
	account, errFind := e.s.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := e.s.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := request.ValidateRevoke(revokeToken)
	if errValidate != nil {
		return errValidate
	}

	account.SetEmail(request.PreviousEmailAddress)
	errUpdate := e.s.accountRepository.Update(db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errDeleteTicket := e.s.updateEmailRepository.DeleteAll(db, account)
	if errDeleteTicket != nil {
		return errDeleteTicket
	}

	// revoke all running sessions
	errRevoke := e.s.sessionRepository.RevokeAll(db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (e *EmailOps) DeleteUpdateRequest(db database.SQLExecutor, account *model.Account) error {
	return e.s.updateEmailRepository.DeleteAll(db, account)
}
