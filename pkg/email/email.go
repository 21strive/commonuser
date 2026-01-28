package email

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
)

type WithTransaction struct {
	EmailOps *EmailOps
	Tx       *sql.Tx
}

func (w *WithTransaction) RequestEmailChange(ctx context.Context, account *model.Account, newEmailAddress string) (*model.UpdateEmail, error) {
	return w.EmailOps.requestEmailChange(ctx, w.Tx, account, newEmailAddress)
}

func (w *WithTransaction) ConfirmEmailChange(ctx context.Context, pipe redis.Pipeliner, account *model.Account, token string) error {
	return w.EmailOps.confirmEmailChange(ctx, pipe, w.Tx, account, token)
}

func (w *WithTransaction) RevokeEmailChange(ctx context.Context, pipe redis.Pipeliner, account *model.Account, revokeToken string) error {
	return w.EmailOps.revokeEmailChange(ctx, pipe, w.Tx, account, revokeToken)
}

func (w *WithTransaction) DeleteEmailChange(ctx context.Context, account *model.Account) error {
	return w.EmailOps.deleteEmailChange(ctx, w.Tx, account)
}

type EmailOps struct {
	writeDB               *sql.DB
	updateEmailRepository *repository.UpdateEmailRepository
	accountOps            *account.AccountOps
	sessionOps            *session.SessionOps
}

func (e *EmailOps) SetWriteDB(db *sql.DB) {
	e.writeDB = db
}

func (e *EmailOps) requestEmailChange(ctx context.Context, db types.SQLExecutor, account *model.Account, newEmailAddress string) (*model.UpdateEmail, error) {

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

func (e *EmailOps) RequestEmailChange(ctx context.Context, account *model.Account, newEmailAddress string) (*model.UpdateEmail, error) {
	return e.requestEmailChange(ctx, e.writeDB, account, newEmailAddress)
}

func (e *EmailOps) confirmEmailChange(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account, token string) error {

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
			errDeleteRequests := e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
			if errDeleteRequests != nil {
				return errDeleteRequests
			}
			return model.EmailChangeRequestExpired
		}
		return errValidate
	}

	request.SetProcessed()
	errUpdateTicket := e.updateEmailRepository.UpdateRequest(ctx, db, request)
	if errUpdateTicket != nil {
		return errUpdateTicket
	}

	// revoke all running sessions
	var errRevoke error
	if pipe != nil {
		errRevoke = e.sessionOps.WithTransaction(pipe, db.(*sql.Tx)).RevokeAll(ctx, account)
	} else {
		errRevoke = e.sessionOps.RevokeAll(ctx, account)
	}
	if errRevoke != nil {
		return errRevoke
	}

	account.SetEmail(request.NewEmailAddress)

	return nil
}

func (e *EmailOps) ConfirmEmailChange(ctx context.Context, account *model.Account, token string) error {
	return e.confirmEmailChange(ctx, nil, e.writeDB, account, token)
}

func (e *EmailOps) revokeEmailChange(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account, revokeToken string) error {

	request, errFind := e.updateEmailRepository.FindRequest(account)
	if errFind != nil {
		return errFind
	}

	errValidate := request.ValidateRevoke(revokeToken)
	if errValidate != nil {
		return errValidate
	}

	account.SetEmail(request.PreviousEmailAddress)

	errDeleteTicket := e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
	if errDeleteTicket != nil {
		return errDeleteTicket
	}

	var errRevoke error
	if pipe != nil {
		errRevoke = e.sessionOps.WithTransaction(pipe, db.(*sql.Tx)).RevokeAll(ctx, account)
	} else {
		errRevoke = e.sessionOps.RevokeAll(ctx, account)
	}

	if errRevoke != nil {
		return errRevoke
	}

	return nil
}

func (e *EmailOps) RevokeEmailChange(ctx context.Context, account *model.Account, revokeToken string) error {
	return e.revokeEmailChange(ctx, nil, e.writeDB, account, revokeToken)
}

func (e *EmailOps) deleteEmailChange(ctx context.Context, db types.SQLExecutor, account *model.Account) error {
	return e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
}

func (e *EmailOps) DeleteEmailChange(ctx context.Context, account *model.Account) error {
	return e.deleteEmailChange(ctx, e.writeDB, account)
}

func New(updateEmailRepository *repository.UpdateEmailRepository, accountOps *account.AccountOps, sessionOps *session.SessionOps) *EmailOps {
	return &EmailOps{
		updateEmailRepository: updateEmailRepository,
		accountOps:            accountOps,
		sessionOps:            sessionOps,
	}
}
