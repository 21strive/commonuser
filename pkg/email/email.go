package email

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/redis/go-redis/v9"
)

type WithTransaction struct {
	EmailOps *EmailOps
	Tx       *sql.Tx
}

func (w *WithTransaction) RequestEmailChange(ctx context.Context, account *model.Account, newEmailAddress string) (*model.UpdateEmail, error) {
	return w.EmailOps.requestEmailChange(ctx, w.Tx, account, newEmailAddress)
}

func (w *WithTransaction) ConfirmEmailChange(ctx context.Context, pipe redis.Pipeliner, accountUUID string, token string) error {
	return w.EmailOps.confirmEmailChange(ctx, pipe, w.Tx, accountUUID, token)
}

func (w *WithTransaction) RevokeEmailChange(ctx context.Context, pipe redis.Pipeliner, accountUUID string, revokeToken string) error {
	return w.EmailOps.revokeEmailChange(ctx, pipe, w.Tx, accountUUID, revokeToken)
}

func (w *WithTransaction) DeleteEmailChange(ctx context.Context, account *model.Account) error {
	return w.EmailOps.deleteEmailChange(ctx, w.Tx, account)
}

type EmailOps struct {
	writeDB               *sql.DB
	updateEmailRepository *repository.UpdateEmailRepository
	accountRepository     *repository.AccountRepository
	sessionRepository     *repository.SessionRepository
}

func (e *EmailOps) Init(updateEmailRepository *repository.UpdateEmailRepository, accountRepository *repository.AccountRepository, sessionRepository *repository.SessionRepository) {
	e.updateEmailRepository = updateEmailRepository
	e.accountRepository = accountRepository
	e.sessionRepository = sessionRepository
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

func (e *EmailOps) confirmEmailChange(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountUUID string, token string) error {
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
			errDeleteRequests := e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
			if errDeleteRequests != nil {
				return errDeleteRequests
			}
			return model.EmailChangeRequestExpired
		}
		return errValidate
	}

	account.SetEmail(request.NewEmailAddress)
	errUpdate := e.accountRepository.Update(ctx, pipe, db, account)
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

func (e *EmailOps) ConfirmEmailChange(ctx context.Context, accountUUID string, token string) error {
	return e.confirmEmailChange(ctx, nil, e.writeDB, accountUUID, token)
}

func (e *EmailOps) revokeEmailChange(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, accountUUID string, revokeToken string) error {
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
	errUpdate := e.accountRepository.Update(ctx, pipe, db, account)
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

func (e *EmailOps) RevokeEmailChange(ctx context.Context, accountUUID string, revokeToken string) error {
	return e.revokeEmailChange(ctx, nil, e.writeDB, accountUUID, revokeToken)
}

func (e *EmailOps) deleteEmailChange(ctx context.Context, db types.SQLExecutor, account *model.Account) error {
	return e.updateEmailRepository.DeleteAllRequest(ctx, db, account)
}

func (e *EmailOps) DeleteEmailChange(ctx context.Context, account *model.Account) error {
	return e.deleteEmailChange(ctx, e.writeDB, account)
}

func New(repositoryPool *database.RepositoryPool, writeDB *sql.DB) *EmailOps {
	return &EmailOps{
		writeDB:               writeDB,
		updateEmailRepository: repositoryPool.UpdateEmailRepository,
		accountRepository:     repositoryPool.AccountRepository,
		sessionRepository:     repositoryPool.SessionRepository,
	}
}
