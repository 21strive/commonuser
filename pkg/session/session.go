package session

import (
	"errors"
	"github.com/21strive/commonuser/internal/database"
	"github.com/21strive/commonuser/internal/fetcher"
	"github.com/21strive/commonuser/internal/model"
	"github.com/21strive/commonuser/internal/repository"
	"time"
)

var InvalidSession = errors.New("invalid session")

type SessionOps struct {
	sessionRepository *repository.SessionRepository
	sessionFetcher    *fetcher.SessionFetcher
}

func (s *SessionOps) Create(db database.SQLExecutor, session *model.Session) error {
	return s.s.sessionRepository.Create(db, session)
}

func (s *SessionOps) Ping(db database.SQLExecutor, sessionRandId string) error {
	sessionFromDB, errFind := s.s.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return errFind
	}

	if sessionFromDB.IsValid() {
		return session.InvalidSession
	}

	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	return s.s.sessionRepository.Update(db, sessionFromDB)
}

func (s *SessionOps) Revoke(db database.SQLExecutor, sessionUUID string) error {
	session, errFind := s.s.sessionRepository.FindByUUID(sessionUUID)
	if errFind != nil {
		return errFind
	}

	session.SetUpdatedAt(time.Now().UTC())
	session.Revoke()
	return s.s.sessionRepository.Update(db, session)
}

func (s *SessionOps) Refresh(db database.SQLExecutor, account *model.Account, sessionRandId string) (string, string, error) {
	sessionFromDB, errFind := s.s.sessionRepository.FindByRandId(sessionRandId)
	if errFind != nil {
		return "", "", errFind
	}
	if !sessionFromDB.IsValid() {
		return "", "", session.InvalidSession
	}

	sessionFromDB.SetUpdatedAt(time.Now().UTC())
	sessionFromDB.SetLastActiveAt(time.Now().UTC())
	sessionFromDB.SetLifeSpan(s.s.config.TokenLifespan)
	sessionFromDB.GenerateRefreshToken()
	errUpdate := s.s.sessionRepository.Update(db, sessionFromDB)
	if errUpdate != nil {
		return "", "", errUpdate
	}

	newAccessToken, errGenerate := account.GenerateAccessToken(s.s.config.JWTSecret, s.s.config.JWTIssuer, s.s.config.JWTLifespan, sessionFromDB.GetRandId())
	if errGenerate != nil {
		return "", "", errGenerate
	}

	return newAccessToken, sessionFromDB.RefreshToken, nil
}

func (s *SessionOps) SeedByAccount(account *model.Account) error {
	return s.s.sessionRepository.SeedByAccount(account)
}

func (s *SessionOps) PurgeInvalid(db database.SQLExecutor) error {
	return s.s.sessionRepository.PurgeInvalid(db)
}

func (s *SessionOps) Ping(sessionRandId string) (*model.Session, error) {
	sessionFromCache, err := s.sessionFetcher.FetchByRandId(sessionRandId)
	if err != nil {
		return nil, err
	}
	if sessionFromCache == nil {
		return nil, constant.Unauthorized
	}
	if !sessionFromCache.IsValid() {
		return nil, constant.Unauthorized
	}

	return sessionFromCache, nil
}

func (s *SessionOps) FetchByAccount(accountRandId string) ([]*model.Session, error) {
	isBlank, errCheck := s.sessionFetcher.IsBlankPage(accountRandId)
	if errCheck != nil {
		return nil, errCheck
	}
	if isBlank {
		return nil, nil
	}

	sessions, err := s.sessionFetcher.FetchByAccount(accountRandId)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, session.SeedRequired
	}

	return sessions, nil
}
