package repository

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type SessionRepository struct {
	base                  *redifu.Base[*model.Session]
	entityName            string
	tableName             string
	findByRandIdStmt      *sql.Stmt
	findByUUIDStmt        *sql.Stmt
	findByAccountUUIDStmt *sql.Stmt
}

func (sm *SessionRepository) GetBase() *redifu.Base[*model.Session] {
	return sm.base
}

func (sm *SessionRepository) Create(ctx context.Context, db database.SQLExecutor, session *model.Session) error {
	tableName := sm.entityName + "_session"
	query := `INSERT INTO ` + tableName + ` (
		uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, user_agent, 
		refresh_token, expired_at, revoked) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := db.ExecContext(ctx, query,
		session.GetUUID(),
		session.GetRandId(),
		session.GetCreatedAt(),
		session.GetUpdatedAt(),
		session.LastActiveAt,
		session.AccountUUID,
		session.DeviceId,
		session.DeviceType,
		session.UserAgent,
		session.RefreshToken,
		session.ExpiredAt,
		session.Revoked)
	if err != nil {
		return err
	}

	return sm.base.Upsert(ctx, session)
}

func (sm *SessionRepository) Update(ctx context.Context, db database.SQLExecutor, session *model.Session) error {
	session.SetUpdatedAt(time.Now().UTC())
	tableName := sm.entityName + "_session"
	query := `UPDATE ` + tableName + ` SET updated_at = $1, last_active_at = $2, 
			  revoked = $3, refresh_token = $4 WHERE uuid = $5`
	_, err := db.ExecContext(ctx,
		query,
		session.GetUpdatedAt(),
		session.LastActiveAt,
		session.Revoked,
		session.RefreshToken,
		session.GetUUID(),
	)
	if err != nil {
		return err
	}

	return sm.base.Upsert(ctx, session)
}

func (sm *SessionRepository) scanSession(ctx context.Context, scanner interface {
	Scan(dest ...interface{}) error
}) (*model.Session, error) {
	session := model.NewSession()
	err := scanner.Scan(
		&session.UUID,
		&session.RandId,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActiveAt,
		&session.AccountUUID,
		&session.DeviceId,
		&session.DeviceType,
		&session.UserAgent,
		&session.RefreshToken,
		&session.ExpiredAt,
		&session.Revoked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.SessionNotFound
		}
		return nil, err
	}

	err = sm.base.Upsert(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *SessionRepository) SessionRowsScanner(rows *sql.Rows) (*model.Session, error) {
	session := model.NewSession()
	err := rows.Scan(
		&session.UUID,
		&session.RandId,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActiveAt,
		&session.AccountUUID,
		&session.DeviceId,
		&session.DeviceType,
		&session.UserAgent,
		&session.RefreshToken,
		&session.ExpiredAt,
		&session.Revoked,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *SessionRepository) FindByRandId(ctx context.Context, randId string) (*model.Session, error) {
	row := sm.findByRandIdStmt.QueryRow(randId)
	return sm.scanSession(ctx, row)
}

func (sm *SessionRepository) FindByUUID(ctx context.Context, uuid string) (*model.Session, error) {
	row := sm.findByUUIDStmt.QueryRow(uuid)
	return sm.scanSession(ctx, row)
}

func (sm *SessionRepository) SeedByRandId(ctx context.Context, randId string) error {
	sessionFromDB, errFind := sm.FindByRandId(ctx, randId)
	if errFind != nil {
		return errFind
	}

	return sm.base.Upsert(ctx, sessionFromDB, randId)
}

func (sm *SessionRepository) RevokeAll(ctx context.Context, db database.SQLExecutor, account *model.Account) error {
	tableName := sm.entityName + "_session"
	query := "UPDATE " + tableName + " SET revoked = true WHERE account_uuid = $1"
	_, errorExec := db.ExecContext(ctx, query, account.GetUUID())
	if errorExec != nil {
		return errorExec
	}

	return nil
}

func (sm *SessionRepository) PurgeInvalid(ctx context.Context, db database.SQLExecutor) error {
	tableName := sm.entityName + "_session"
	query := "DELETE FROM " + tableName + " WHERE expired_at < NOW() AND revoked = true"
	_, errorExec := db.ExecContext(ctx, query)
	if errorExec != nil {
		return errorExec
	}

	return nil
}

func NewSessionRepository(readDB *sql.DB, redis redis.UniversalClient, app *config.App) *SessionRepository {
	tableName := app.EntityName + "_session"
	findByRandIdStmt, errPrepare := readDB.Prepare(`SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, 
       				 user_agent, refresh_token, expired_at, revoked FROM ` + tableName + ` WHERE randid = $1`)
	findManyByAccountStmt, errPrepare := readDB.Prepare(`SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, 
       				 user_agent, refresh_token, expired_at, revoked FROM ` + tableName + ` WHERE account_uuid = $1`)
	findByUUIDStmt, errPrepare := readDB.Prepare(`SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, 
       				 user_agent, refresh_token, expired_at, revoked FROM ` + tableName + ` WHERE uuid = $1`)
	if errPrepare != nil {
		panic(errPrepare)
	}

	base := redifu.NewBase[*model.Session](redis, app.EntityName+":session:%s", app.TokenLifespan)

	return &SessionRepository{
		base:                  base,
		entityName:            app.EntityName,
		findByRandIdStmt:      findByRandIdStmt,
		findByUUIDStmt:        findByUUIDStmt,
		findByAccountUUIDStmt: findManyByAccountStmt,
	}
}
