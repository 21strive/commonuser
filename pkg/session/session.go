package session

import (
	"context"
	"github.com/21strive/commonuser/config"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"time"
)

type SessionOps struct {
	sessionRepository *repository.SessionRepository
	sessionFetcher    *fetcher.SessionFetcher
	config            *config.App
}

func (s *SessionOps) Init(
	sessionRepository *repository.SessionRepository,
	sessionFetcher *fetcher.SessionFetcher,
	config *config.App,
) {
	s.sessionRepository = sessionRepository
	s.sessionFetcher = sessionFetcher
	s.config = config
}

func (s *SessionOps) Create(ctx context.Context, db database.SQLExecutor, session *model.Session) error {
	return s.sessionRepository.Create(ctx, db, session)
}

func (s *SessionOps) Ping(ctx context.Context, db database.SQLExecutor, sessionRandId string) error {
	sessionFromDB, errFind := s.sessionRepository.FindByRandId(ctx, sessionRandId)
	if errFind != nil {
		return errFind
	}

	if sessionFromDB.IsValid() {
		return model.InvalidSession
	}

	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	return s.sessionRepository.Update(ctx, db, sessionFromDB)
}

func (s *SessionOps) Revoke(ctx context.Context, db database.SQLExecutor, sessionUUID string) error {
	session, errFind := s.sessionRepository.FindByUUID(ctx, sessionUUID)
	if errFind != nil {
		return errFind
	}

	session.SetUpdatedAt(time.Now().UTC())
	session.Revoke()
	return s.sessionRepository.Update(ctx, db, session)
}

func (s *SessionOps) Refresh(ctx context.Context, db database.SQLExecutor, account *model.Account, sessionRandId string) (string, string, error) {
	sessionFromDB, errFind := s.sessionRepository.FindByRandId(ctx, sessionRandId)
	if errFind != nil {
		return "", "", errFind
	}
	if !sessionFromDB.IsValid() {
		return "", "", model.InvalidSession
	}

	sessionFromDB.SetUpdatedAt(time.Now().UTC())
	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	sessionFromDB.SetLifeSpan(s.config.TokenLifespan)
	sessionFromDB.GenerateRefreshToken()
	errUpdate := s.sessionRepository.Update(ctx, db, sessionFromDB)
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

func (s *SessionOps) PurgeInvalid(ctx context.Context, db database.SQLExecutor) error {
	return s.sessionRepository.PurgeInvalid(ctx, db)
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

func New() *SessionOps {
	return &SessionOps{}
}
