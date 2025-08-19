package postgresql

import (
	"database/sql"
	"github.com/21strive/commonuser/lib"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
)

type SessionSQL struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	lib.Session
}

func NewSessionSQL() *SessionSQL {
	session := &SessionSQL{}
	redifu.InitSQLItem(session)
	session.IsActive = true // active by default
	return session
}

type SessionManagerSQL struct {
	base       *redifu.Base[SessionSQL]
	db         *sql.DB
	entityName string
}

func (sm *SessionManagerSQL) Create(session SessionSQL) error {
	tableName := sm.entityName + "_session"
	query := `INSERT INTO ` + tableName + ` (uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := sm.db.Exec(query,
		session.GetUUID(),
		session.GetRandId(),
		session.GetCreatedAt(),
		session.GetUpdatedAt(),
		session.LastActiveAt,
		session.AccountUUID,
		session.DeviceId,
		session.DeviceInfo,
		session.UserAgent,
		session.RefreshToken,
		session.ExpiresAt,
		session.IsActive)
	if err != nil {
		return err
	}

	return sm.base.Set(session, session.RefreshToken)
}

func (sm *SessionManagerSQL) Update(session *SessionSQL) error {
	tableName := sm.entityName + "_session"
	query := `UPDATE ` + tableName + ` SET last_active_at = $1, is_active = $2, deactivated_at = $3 WHERE uuid = $4`
	_, err := sm.db.Exec(
		query,
		session.LastActiveAt,
		session.IsActive,
		session.DeactivatedAt,
		session.GetUUID(),
	)
	if err != nil {
		return err
	}

	return sm.base.Set(*session, session.RefreshToken)
}

func (sm *SessionManagerSQL) scanSession(scanner interface {
	Scan(dest ...interface{}) error
}) (*SessionSQL, error) {
	session := NewSessionSQL()
	redifu.InitSQLItem(session)
	err := scanner.Scan(
		&session.UUID,
		&session.RandId,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastActiveAt,
		&session.AccountUUID,
		&session.DeviceId,
		&session.DeviceInfo,
		&session.UserAgent,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.IsActive,
		&session.DeactivatedAt,
	)
	if err != nil {
		return nil, err
	}

	err = sm.base.Set(*session, session.RefreshToken)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *SessionManagerSQL) FindByHash(hash string) (*SessionSQL, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, deactivated_at FROM ` + tableName + ` WHERE refresh_token = $1`
	row := sm.db.QueryRow(query, hash)

	return sm.scanSession(row)
}

func (sm *SessionManagerSQL) FindByRandId(randId string) (*SessionSQL, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, deactivated_at FROM ` + tableName + ` WHERE randid = $1`
	row := sm.db.QueryRow(query, randId)

	return sm.scanSession(row)
}

func (sm *SessionManagerSQL) FindByAccountUUID(accountUUID string) ([]SessionSQL, error) {
	tableName := sm.entityName + "_session"
	query := `SELECT uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_info, user_agent, refresh_token, expires_at, is_active, deactivated_at FROM ` + tableName + ` WHERE account_uuid = $1`
	rows, err := sm.db.Query(query, accountUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionSQL
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

func NewSessionManagerSQL(db *sql.DB, redis redis.UniversalClient, entityName string) *SessionManagerSQL {
	base := redifu.NewBase[SessionSQL](redis, entityName+":session:%s")
	return &SessionManagerSQL{
		base:       base,
		db:         db,
		entityName: entityName,
	}
}
