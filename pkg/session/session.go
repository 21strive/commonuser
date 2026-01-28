package session

import (
	"context"
	"database/sql"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"github.com/21strive/commonuser/internal/types"
	"github.com/21strive/redifu"
	"github.com/redis/go-redis/v9"
	"time"
)

type WithTranscation struct {
	SessionOps *SessionOps
	Tx         *sql.Tx
	pipe       redis.Pipeliner
}

func (w *WithTranscation) Create(ctx context.Context, session *model.Session) error {
	return w.SessionOps.create(ctx, w.pipe, w.Tx, session)
}

func (w *WithTranscation) Ping(ctx context.Context, sessionRandId string) error {
	return w.SessionOps.ping(ctx, w.pipe, w.Tx, sessionRandId)
}

func (w *WithTranscation) Revoke(ctx context.Context, sessionUUID string) error {
	return w.SessionOps.revoke(ctx, w.pipe, w.Tx, sessionUUID)
}

func (w *WithTranscation) RevokeAll(ctx context.Context, account *model.Account) error {
	return w.SessionOps.revokeAll(ctx, w.pipe, w.Tx, account)
}

func (w *WithTranscation) Refresh(ctx context.Context, account *model.Account, sessionRandId string) (string, string, error) {
	return w.SessionOps.refresh(ctx, w.pipe, w.Tx, account, sessionRandId)
}

func (w *WithTranscation) PurgeInvalid(ctx context.Context) error {
	return w.SessionOps.purgeInvalid(ctx, w.Tx)
}

type SessionOps struct {
	writeDB           *sql.DB
	sessionRepository *repository.SessionRepository
	sessionFetcher    *fetcher.SessionFetcher
	config            *config.App
}

func (s *SessionOps) SetWriteDB(db *sql.DB) {
	s.writeDB = db
}

func (s *SessionOps) GetSessionBase() *redifu.Base[*model.Session] {
	return s.sessionRepository.GetBase()
}

func (s *SessionOps) WithTransaction(pipe redis.Pipeliner, tx *sql.Tx) *WithTranscation {
	return &WithTranscation{SessionOps: s, pipe: pipe, Tx: tx}
}

func (s *SessionOps) create(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, session *model.Session) error {
	return s.sessionRepository.Create(ctx, pipe, db, session)
}

func (s *SessionOps) Create(ctx context.Context, session *model.Session) error {
	return s.create(ctx, nil, s.writeDB, session)
}

func (s *SessionOps) ping(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, sessionRandId string) error {
	sessionFromDB, errFind := s.sessionRepository.FindByRandId(ctx, pipe, sessionRandId)
	if errFind != nil {
		return errFind
	}

	if sessionFromDB.IsValid() {
		return model.InvalidSession
	}

	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	return s.sessionRepository.Update(ctx, pipe, db, sessionFromDB)
}

func (s *SessionOps) Ping(ctx context.Context, sessionRandId string) error {
	return s.ping(ctx, nil, s.writeDB, sessionRandId)
}

func (s *SessionOps) revoke(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, sessionUUID string) error {
	session, errFind := s.sessionRepository.FindByUUID(ctx, pipe, sessionUUID)
	if errFind != nil {
		return errFind
	}

	session.SetUpdatedAt(time.Now().UTC())
	session.Revoke()
	return s.sessionRepository.Update(ctx, pipe, db, session)
}

func (s *SessionOps) Revoke(ctx context.Context, sessionUUID string) error {
	return s.revoke(ctx, nil, s.writeDB, sessionUUID)
}

func (s *SessionOps) revokeAll(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account) error {
	sessions, errFind := s.sessionRepository.FindManyByAccount(ctx, nil, account.GetUUID())
	if errFind != nil {
		return errFind
	}

	for _, session := range sessions {
		session.SetUpdatedAt(time.Now().UTC())
		session.Revoke()
		errUpdate := s.sessionRepository.Update(ctx, pipe, db, session)
		if errUpdate != nil {
			return errUpdate
		}
	}

	return nil
}

func (s *SessionOps) RevokeAll(ctx context.Context, account *model.Account) error {
	return s.revokeAll(ctx, nil, s.writeDB, account)
}

func (s *SessionOps) refresh(ctx context.Context, pipe redis.Pipeliner, db types.SQLExecutor, account *model.Account, sessionRandId string) (string, string, error) {
	sessionFromDB, errFind := s.sessionRepository.FindByRandId(ctx, pipe, sessionRandId)
	if errFind != nil {
		return "", "", errFind
	}
	if !sessionFromDB.IsValid() {
		return "", "", model.InvalidSession
	}

	sessionFromDB.SetUpdatedAt(time.Now().UTC())
	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	sessionFromDB.SetLifeSpan(s.config.TokenLifespan)
	errGenerate := sessionFromDB.GenerateRefreshToken()
	if errGenerate != nil {
		return "", "", errGenerate
	}
	errUpdate := s.sessionRepository.Update(ctx, pipe, db, sessionFromDB)
	if errUpdate != nil {
		return "", "", errUpdate
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(
		s.config.JWTSecret,
		s.config.JWTIssuer,
		s.config.JWTLifespan,
		sessionFromDB.GetRandId(),
	)
	if errGenerate != nil {
		return "", "", errGenerate
	}

	return newAccessToken, sessionFromDB.RefreshToken, nil
}

func (s *SessionOps) Refresh(ctx context.Context, account *model.Account, sessionRandId string) (string, string, error) {
	return s.refresh(ctx, nil, s.writeDB, account, sessionRandId)
}

func (s *SessionOps) purgeInvalid(ctx context.Context, db types.SQLExecutor) error {
	return s.sessionRepository.PurgeInvalid(ctx, db)
}

func (s *SessionOps) PurgeInvalid(ctx context.Context) error {
	return s.purgeInvalid(ctx, s.writeDB)
}

func (s *SessionOps) PingByCache(ctx context.Context, sessionRandId string) (*model.Session, error) {
	sessionFromCache, err := s.sessionFetcher.FetchByRandId(ctx, sessionRandId)
	if err != nil {
		return nil, err
	}
	if sessionFromCache == nil {
		return nil, model.Unauthorized
	}
	if !sessionFromCache.IsValid() {
		return nil, model.Unauthorized
	}

	return sessionFromCache, nil
}

func New(sessionRepository *repository.SessionRepository, sessionFetcher *fetcher.SessionFetcher, config *config.App) *SessionOps {
	return &SessionOps{
		sessionRepository: sessionRepository,
		sessionFetcher:    sessionFetcher,
		config:            config,
	}
}
