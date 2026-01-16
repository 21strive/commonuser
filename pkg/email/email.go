package email

import (
	"context"
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
)

type EmailOps struct {
	updateEmailRepository *repository.EmailRepository
	accountRepository     *repository.AccountRepository
	sessionRepository     *repository.SessionRepository
}

func (e *EmailOps) Init(updateEmailRepository *repository.EmailRepository, accountRepository *repository.AccountRepository, sessionRepository *repository.SessionRepository) {
	e.updateEmailRepository = updateEmailRepository
	e.accountRepository = accountRepository
	e.sessionRepository = sessionRepository
}

func (e *EmailOps) RequestEmailChange(ctx context.Context, db database.SQLExecutor, account *model.Account, newEmailAddress string) (*model.UpdateEmail, error) {

	requestFromDB, errFind := e.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		if !errors.Is(errFind, model.EmailChangeTokenNotFound) {
			return nil, errFind
		}
	}
	if requestFromDB != nil {
		if requestFromDB.IsExpired() {
			errDeleteRequest := e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
			if errDeleteRequest != nil {
				return nil, errDeleteRequest
			}
		} else {
			return requestFromDB, nil
		}
	}

	updateEmailRequest := model.NewUpdateEmailRequest()
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

	errCreateTicket := e.updateEmailRepository.CreateRequest(ctx, db, updateEmailRequest)
	if errCreateTicket != nil {
		return nil, errCreateTicket
	}

	return updateEmailRequest, nil
}

func (e *EmailOps) ConfirmEmailChange(ctx context.Context, db database.SQLExecutor, accountUUID string, token string) error {
	account, errFind := e.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := e.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}
	if request.Processed {
		return nil
	}

	errValidate := request.Validate(token)
	if errValidate != nil {
		if errors.Is(errValidate, model.EmailChangeRequestExpired) {
			e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
			return model.EmailChangeRequestExpired
		}
		return errValidate
	}

	account.SetEmail(request.NewEmailAddress)
	errUpdate := e.accountRepository.Update(ctx, db, account)
	if errUpdate != nil {
		return errUpdate
	}

	request.SetProcessed()
	errUpdateTicket := e.updateEmailRepository.UpdateRequest(ctx, db, request)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	errRevoke := e.sessionRepository.RevokeAll(ctx, db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (e *EmailOps) RevokeEmailChange(ctx context.Context, db database.SQLExecutor, accountUUID string, revokeToken string) error {
	account, errFind := e.accountRepository.FindByUUID(accountUUID)
	if errFind != nil {
		return errFind
	}

	request, errFind := e.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := request.ValidateRevoke(revokeToken)
	if errValidate != nil {
		return errValidate
	}

	account.SetEmail(request.PreviousEmailAddress)
	errUpdate := e.accountRepository.Update(ctx, db, account)
	if errUpdate != nil {
		return errUpdate
	}

	errDeleteTicket := e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
	if errDeleteTicket != nil {
		return errDeleteTicket
	}

	// revoke all running sessions
	errRevoke := e.sessionRepository.RevokeAll(ctx, db, account)
	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (e *EmailOps) DeleteEmailChange(ctx context.Context, db database.SQLExecutor, account *model.Account) error {
	return e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
}

func New() *EmailOps {
	return &EmailOps{}
}
