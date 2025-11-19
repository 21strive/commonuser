package session

import (
	"database/sql"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type Repository struct {
	base                   *redifu.Base[*Session]
	sessionByAccount       *redifu.Sorted[*Session]
	sessionByAccountSeeder *redifu.SortedSeeder[*Session]
	entityName             string
	tableName              string
	findByRandIdStmt       *sql.Stmt
	findByUUIDStmt         *sql.Stmt
	findByAccountUUIDStmt  *sql.Stmt
}

func (sm *Repository) GetBase() *redifu.Base[*Session] {
	return sm.base
}

func (sm *Repository) Create(db shared.SQLExecutor, session *Session) error {
	tableName := sm.entityName + "_session"
	query := `INSERT INTO ` + tableName + ` (
		uuid, randid, created_at, updated_at, last_active_at, account_uuid, device_id, device_type, user_agent, 
		refresh_token, expired_at, revoked) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
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
		session.ExpiredAt,
		session.Revoked)
	if err != nil {
		return err
	}

	return sm.base.Set(session)
}

func (sm *Repository) Update(db shared.SQLExecutor, session *Session) error {
	session.SetUpdatedAt(time.Now().UTC())
	tableName := sm.entityName + "_session"
	query := `UPDATE ` + tableName + ` SET updated_at = $1, last_active_at = $2, 
			  revoked = $3, refresh_token = $4 WHERE uuid = $5`
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

	return sm.base.Set(session)
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
		&session.ExpiredAt,
		&session.Revoked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, SessionNotFound
		}
		return nil, err
	}

	err = sm.base.Set(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (sm *Repository) SessionRowsScanner(rows *sql.Rows) (*Session, error) {
	session := NewSession()
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

func (sm *Repository) FindByRandId(randId string) (*Session, error) {
	row := sm.findByRandIdStmt.QueryRow(randId)
	return sm.scanSession(row)
}

func (sm *Repository) FindByUUID(uuid string) (*Session, error) {
	row := sm.findByUUIDStmt.QueryRow(uuid)
	return sm.scanSession(row)
}

func (sm *Repository) SeedByRandId(randId string) error {
	sessionFromDB, errFind := sm.FindByRandId(randId)
	if errFind != nil {
		return errFind
	}

	return sm.base.Set(sessionFromDB, randId)
}

func (sm *Repository) SeedByAccount(account *account.Account) error {
	tableName := sm.entityName + "_session"
	query := "SELECT * FROM " + tableName + " WHERE account_uuid = $1"
	return sm.sessionByAccountSeeder.Seed(query, sm.SessionRowsScanner, []interface{}{account.GetUUID()}, []string{account.GetRandId()})
}

func (sm *Repository) RevokeAll(db shared.SQLExecutor, account *account.Account) error {
	tableName := sm.entityName + "_session"
	query := "UPDATE " + tableName + " SET revoked = true WHERE account_uuid = $1"
	_, errorExec := db.Exec(query, account.GetUUID())
	if errorExec != nil {
		return errorExec
	}

	return nil
}

func (sm *Repository) PurgeInvalid(db shared.SQLExecutor) error {
	tableName := sm.entityName + "_session"
	query := "DELETE FROM " + tableName + " WHERE expired_at < NOW() AND revoked = true"
	_, errorExec := db.Exec(query)
	if errorExec != nil {
		return errorExec
	}

	return nil
}

func NewRepository(readDB *sql.DB, redis redis.UniversalClient, app *config.App) *Repository {
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

	base := redifu.NewBase[*Session](redis, app.EntityName+":session:%s", app.TokenLifespan)
	sortedSession := redifu.NewSorted[*Session](redis, base, app.EntityName+":session:account:%s", app.PaginationAge)
	sortedSessionSeeder := redifu.NewSortedSeeder[*Session](readDB, base, sortedSession)

	return &Repository{
		base:                   base,
		sessionByAccount:       sortedSession,
		sessionByAccountSeeder: sortedSessionSeeder,
		entityName:             app.EntityName,
		findByRandIdStmt:       findByRandIdStmt,
		findByUUIDStmt:         findByUUIDStmt,
		findByAccountUUIDStmt:  findManyByAccountStmt,
	}
}
