package postgresql

import (
	"database/sql"
	"github.com/21strive/commonuser/lib"
	"github.com/21strive/redifu"
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
	tableName := sm.entityName + "session"
	query := `INSERT INTO ` + tableName + ` (uuid, randId, createdat, updatedat, lastactiveat, accountuuid, deviceid, deviceinfo, useragent, refreshtoken, expiresat, isrevoked) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
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
	tableName := sm.entityName + "session"
	query := `UPDATE ` + tableName + ` SET lastactiveat =$1, isrevoked = $2, deactivatedat = $3 WHERE uuid = $4`
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

func (sm *SessionManagerSQL) FindByHash(hash string) (*SessionSQL, error) {
	tableName := sm.entityName + "session"
	query := `SELECT uuid, randId, createdat, updatedat, lastactiveat, accountuuid, deviceid, deviceinfo, useragent, refreshtoken, expiresat, isrevoked FROM ` + tableName + ` WHERE refreshtoken = $1`
	row := sm.db.QueryRow(query, hash)

	session := NewSessionSQL()
	redifu.InitSQLItem(session)
	err := row.Scan(
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

func (sm *SessionManagerSQL) FindByAccountUUID(accountUUID string) ([]SessionSQL, error) {
	tableName := sm.entityName + "session"
	query := `SELECT uuid, randId, createdat, updatedat, lastactiveat, accountuuid, deviceid, deviceinfo, useragent, refreshtoken, expiresat, isrevoked FROM ` + tableName + ` WHERE accountuuid = $1`
	rows, err := sm.db.Query(query, accountUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionSQL
	for rows.Next() {
		session := NewSessionSQL()
		redifu.InitSQLItem(session)
		err := rows.Scan(
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
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}
