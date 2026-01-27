package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/types"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type AccountRepository struct {
	redis              redis.UniversalClient
	base               *redifu.Base[*model.Account]
	baseReference      *redifu.Base[*model.AccountReference]
	findByUsernameStmt *sql.Stmt
	findByRandIdStmt   *sql.Stmt
	findByEmailStmt    *sql.Stmt
	findByUUIDStmt     *sql.Stmt
	app                *config.App
}

func (ar *AccountRepository) GetBase() *redifu.Base[*model.Account] {
	return ar.base
}

func (ar *AccountRepository) Close() {
	ar.findByUsernameStmt.Close()
	ar.findByRandIdStmt.Close()
	ar.findByEmailStmt.Close()
	ar.findByUUIDStmt.Close()
}

func (ar *AccountRepository) Create(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	query := "INSERT INTO " + ar.app.EntityName + ` (
		uuid, 
		randid, 
		created_at, 
		updated_at, 
		name, 
		username, 
		password, 
		email, 	
		avatar
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, errInsert := db.ExecContext(ctx,
		query,
		account.GetUUID(),
		account.GetRandId(),
		account.GetCreatedAt(),
		account.GetUpdatedAt(),
		account.Name,
		account.Username,
		account.Password,
		account.Email,
		account.Avatar,
	)

	if errInsert != nil {
		return errInsert
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
		selfPipe = true
	}

	errSetAcc := ar.base.WithPipeline(pipe).Set(ctx, account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) Update(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	query := "UPDATE " + ar.app.EntityName +
		" SET updated_at = $1, name = $2, username = $3, password = $4, email = $5, avatar = $6, email_verified = $7 WHERE uuid = $8"
	_, errUpdate := db.ExecContext(ctx,
		query,
		account.GetUpdatedAt(),
		account.Name,
		account.Username,
		account.Password,
		account.Email,
		account.Avatar,
		account.EmailVerified,
		account.GetUUID())
	if errUpdate != nil {
		return errUpdate
	}

	var errSetAcc error
	if pipe == nil {
		errSetAcc = ar.base.Set(ctx, account)
	} else {
		errSetAcc = ar.base.WithPipeline(pipe).Set(ctx, account)
	}

	if errSetAcc != nil {
		return errSetAcc
	}

	return nil
}

func (ar *AccountRepository) UpdateReference(ctx context.Context, pipe redis.Pipeliner, account *model.Account, oldUsername string, newUsername string) error {
	ref, errGet := ar.baseReference.Get(ctx, oldUsername)
	if errGet != nil {
		return errGet
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
	}

	err := ar.baseReference.WithPipeline(pipe).Del(ctx, ref)
	if err != nil {
		return err
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSet := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, newUsername)
	if errSet != nil {
		return errSet
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) Delete(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	query := "DELETE FROM " + ar.app.EntityName + " WHERE uuid = $1"
	_, errDelete := db.ExecContext(ctx, query, account.GetUUID())
	if errDelete != nil {
		return errDelete
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
	}

	errDelAcc := ar.base.WithPipeline(pipe).Del(ctx, account)
	if errDelAcc != nil {
		return errDelAcc
	}

	referenceExists := true
	ref, errGet := ar.baseReference.Get(ctx, account.Username)
	if errGet != nil {
		if errGet == redis.Nil {
			referenceExists = false
		} else {
			return errGet
		}
	}
	if referenceExists {
		err := ar.baseReference.WithPipeline(pipe).Del(ctx, ref)
		if err != nil {
			return err
		}
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) FindByUsername(username string) (*model.Account, error) {
	return AccountRowScanner(ar.findByUsernameStmt.QueryRow(username))
}

func (ar *AccountRepository) SeedByUsername(ctx context.Context, pipe redis.Pipeliner, username string) error {
	account, err := ar.FindByUsername(username)
	if err != nil {
		return err
	}
	if account == nil {
		errSetBlank := ar.baseReference.MarkAsMissing(ctx, username)
		if errSetBlank != nil {
			return errSetBlank
		}

		return model.AccountDoesNotExists
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
		selfPipe = true
	}

	errSetAcc := ar.base.WithPipeline(pipe).Set(ctx, account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) FindByRandId(randId string) (*model.Account, error) {
	return AccountRowScanner(ar.findByRandIdStmt.QueryRow(randId))
}

func (ar *AccountRepository) SeedByRandId(ctx context.Context, pipe redis.Pipeliner, randId string) error {
	account, err := ar.FindByRandId(randId)
	if err != nil {
		if errors.Is(err, model.AccountDoesNotExists) {
			errSetBlank := ar.baseReference.MarkAsMissing(ctx, randId)
			if errSetBlank != nil {
				return errSetBlank
			}
		}
		return err
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
		selfPipe = true
	}

	errSetAcc := ar.base.WithPipeline(pipe).Set(ctx, account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) FindByEmail(email string) (*model.Account, error) {
	return AccountRowScanner(ar.findByEmailStmt.QueryRow(email))
}

func (ar *AccountRepository) SeedByEmail(ctx context.Context, pipe redis.Pipeliner, email string) error {
	account, err := ar.FindByEmail(email)
	if err != nil {
		return err
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
		selfPipe = true
	}

	errSetAcc := ar.base.WithPipeline(pipe).Set(ctx, account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func (ar *AccountRepository) FindByUUID(uuid string) (*model.Account, error) {
	return AccountRowScanner(ar.findByUUIDStmt.QueryRow(uuid))
}

func (ar *AccountRepository) SeedByUUID(ctx context.Context, pipe redis.Pipeliner, uuid string) error {
	account, err := ar.FindByUUID(uuid)
	if err != nil {
		return err
	}

	var selfPipe bool
	if pipe == nil {
		pipe = ar.redis.Pipeline()
		selfPipe = true
	}

	errSetAcc := ar.base.WithPipeline(pipe).Set(ctx, account)
	if errSetAcc != nil {
		return errSetAcc
	}

	accountReference := model.NewReference()
	accountReference.SetAccountRandId(account.GetRandId())
	errSetReference := ar.baseReference.WithPipeline(pipe).Set(ctx, accountReference, account.Username)
	if errSetReference != nil {
		return errSetReference
	}

	if selfPipe {
		_, errExec := pipe.Exec(ctx)
		return errExec
	}

	return nil
}

func AccountRowScanner(row *sql.Row) (*model.Account, error) {
	account := model.NewAccount()
	err := row.Scan(
		&account.UUID,
		&account.RandId,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.Base.Name,
		&account.Base.Username,
		&account.Base.Password,
		&account.Base.Email,
		&account.Base.Avatar,
		&account.Base.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.AccountDoesNotExists
		}
		return nil, err
	}

	return account, nil
}

func NewAccountRepository(readDB *sql.DB, redis redis.UniversalClient, baseAccount *redifu.Base[*model.Account], baseReference *redifu.Base[*model.AccountReference], app *config.App) *AccountRepository {
	var errPrepare error
	findByUsernameStmt, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE username = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByRandId, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE randId = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByEmailStmt, errPrepare := readDB.Prepare("" +
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
		app.EntityName + " WHERE email = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}
	findByUUIDStmt, errPrepare := readDB.Prepare(
		"SELECT uuid, randid, created_at, updated_at, name, username, password, email, avatar, email_verified FROM " +
			app.EntityName + " WHERE uuid = $1")
	if errPrepare != nil {
		panic(errPrepare)
	}

	return &AccountRepository{
		base:               baseAccount,
		baseReference:      baseReference,
		redis:              redis,
		findByUsernameStmt: findByUsernameStmt,
		findByRandIdStmt:   findByRandId,
		findByEmailStmt:    findByEmailStmt,
		findByUUIDStmt:     findByUUIDStmt,
		app:                app,
	}
}
