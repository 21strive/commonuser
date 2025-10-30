package session

import (
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type Repository struct {
	base       *redifu.Base[*Session]
	db         *sql.DB
	entityName string
}

func (sm *Repository) GetBase() *redifu.Base[*Session] {
	return sm.base
}

func (sm *Repository) Create(db shared.SQLExecutor, session *Session) error {
	tableName := sm.entityName + "_session"
	query := `INSERT INTO ` + tableName + ` (uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, user_agent, refresh_token, expires_at, revoked) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := db.Exec(query,
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
		session.ExpiresAt,
		session.Revoked)
	if err != nil {
		return err
	}

	return sm.base.Set(session, session.RefreshToken)
}

func (sm *Repository) Update(db shared.SQLExecutor, session *Session) error {
	session.SetUpdatedAt(time.Now().UTC())
	tableName := sm.entityName + "_session"
	query := `UPDATE ` + tableName + ` SET updated_at = $1, last_active_at = $2, revoked = $3, refresh_token = $4 WHERE uuid = $5`
	_, err := db.Exec(
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

	return sm.base.Set(session, session.RefreshToken)
}

func (sm *Repository) scanSession(scanner interface {
	Scan(dest ...interface{}) error
}) (*Session, error) {
	session := NewSession()
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
		&session.ExpiresAt,
		&session.Revoked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, SessionNotFound
		}
		return nil, err
	}

	err = sm.base.Set(session, session.RefreshToken)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *Repository) FindByHash(hash string) (*Session, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, COALESCE(deactivated_at, '1970-01-01 00:00:00'::timestamp) FROM ` + tableName + ` WHERE refresh_token = $1`
	row := sm.db.QueryRow(query, hash)

	return sm.scanSession(row)
}

func (sm *Repository) FindByRandId(randId string) (*Session, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, COALESCE(deactivated_at, '1970-01-01 00:00:00'::timestamp) FROM ` + tableName + ` WHERE randid = $1`
	row := sm.db.QueryRow(query, randId)

	return sm.scanSession(row)
}

func (sm *Repository) SeedByRandId(randId string) error {
	sessionFromDB, errFind := sm.FindByRandId(randId)
	if errFind != nil {
		return errFind
	}

	return sm.base.Set(sessionFromDB, randId)
}

func (sm *Repository) FindByAccountUUID(accountUUID string) ([]Session, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, COALESCE(deactivated_at, '1970-01-01 00:00:00'::timestamp) FROM ` + tableName + ` WHERE account_uuid = $1`
	rows, err := sm.db.Query(query, accountUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {

		session, errScan := sm.scanSession(rows)
		if errScan != nil {
			return nil, errScan
		}
		sessions = append(sessions, *session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func NewRepository(readDB *sql.DB, redis redis.UniversalClient, app *config.App) *Repository {
	base := redifu.NewBase[*Session](redis, app.EntityName+":session:%s", app.RecordAge)
	return &Repository{
		base:       base,
		db:         readDB,
		entityName: app.EntityName,
	}
}
