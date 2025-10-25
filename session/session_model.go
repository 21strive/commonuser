package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/21strive/redifu"
	"time"
)

type Session struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	LastActiveAt    time.Time `json:"lastloginat"`
	AccountUUID     string    `json:"accountuuid"`
	DeviceId        string    `json:"deviceid"`
	DeviceInfo      string    `json:"deviceinfo"`
	UserAgent       string    `json:"useragent"`
	RefreshToken    string    `json:"refreshToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
	IsActive        bool      `json:"revoked"`
	DeactivatedAt   time.Time `json:"deactivatedAt"`
}

func (s *Session) SetLastActiveAt(lastActiveAt time.Time) {
	s.LastActiveAt = lastActiveAt
}

func (s *Session) SetAccountUUID(accountUUID string) {
	s.AccountUUID = accountUUID
}

func (s *Session) SetDeviceId(deviceId string) {
	s.DeviceId = deviceId
}

func (s *Session) SetDeviceInfo(deviceInfo string) {
	s.DeviceInfo = deviceInfo
}

func (s *Session) SetUserAgent(userAgent string) {
	s.UserAgent = userAgent
}

func (s *Session) GenerateRefreshToken() error {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return err
	}

	refreshToken := hex.EncodeToString(bytes)
	hashedRefreshTokenBytes := sha256.Sum256([]byte(refreshToken))
	s.RefreshToken = hex.EncodeToString(hashedRefreshTokenBytes[:])
	return nil
}

func (s *Session) SetLifeSpan(refreshTokenLifeSpan time.Duration) {
	timeNow := time.Now().UTC()
	expiresAt := timeNow.Add(refreshTokenLifeSpan)
	s.ExpiresAt = expiresAt
}

func (s *Session) Deactivate() {
	s.IsActive = false
	s.DeactivatedAt = time.Now().UTC()
}

func (s *Session) MarkActivity() {
	s.LastActiveAt = time.Now().UTC()
}

func (s *Session) IsValid() bool {
	if s.ExpiresAt.Before(time.Now().UTC()) {
		return false
	}
	if !s.IsActive {
		return false
	}
	return true
}

func NewSession() *Session {
	session := &Session{}
	redifu.InitSQLItem(session)
	session.IsActive = true // active by default
	return session
}
