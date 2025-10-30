package account

import (
	"github.com/21strive/redifu"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matthewhartstonge/argon2"
	"time"
)

type UserClaims struct {
	UUID      string `json:"uuid"` // user uuid
	RandId    string `json:"randId"`
	Name      string `json:"name"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
	SessionID string `json:"sessionid"`
	jwt.RegisteredClaims
}

type AssociatedAccount struct {
	Name     string `json:"name,omitempty" db:"-"`
	Email    string `json:"email,omitempty" db:"-"`
	Uuid     string `json:"uuid,omitempty" db:"-"`
	Sub      string `json:"sub,omitempty" db:"-"`
	Provider string `json:"provider,omitempty" db:"-"`
}

type Base struct {
	Name              string              `json:"name,omitempty" db:"name"`
	Username          string              `json:"username,omitempty" db:"username"`
	Password          string              `json:"-" db:"password"`
	Email             string              `json:"email,omitempty" db:"email"`
	Avatar            string              `json:"avatar,omitempty" db:"avatar"`
	EmailVerified     bool                `json:"email_verified,omitempty" db:"email_verified"`
	AssociatedAccount []AssociatedAccount `json:"associatedAccount,omitempty" db:"-"`
}

func (b *Base) SetName(name string) {
	b.Name = name
}

func (b *Base) SetUsername(username string) {
	b.Username = username
}

func (b *Base) SetPassword(password string) error {
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(password))
	if err != nil {
		return err
	}

	b.Password = string(encoded)
	return nil
}

func (b *Base) VerifyPassword(password string) (bool, error) {
	match, err := argon2.VerifyEncoded([]byte(password), []byte(b.Password))
	if err != nil {
		return false, err
	}
	if match {
		return true, nil
	}
	return false, nil
}

func (b *Base) SetEmail(email string) {
	b.Email = email
}

func (b *Base) SetEmailVerified() {
	b.EmailVerified = true
}

func (b *Base) SetAvatar(avatar string) {
	b.Avatar = avatar
}

func (b *Base) SetAssociatedAccount(associatedAccount AssociatedAccount) {
	b.AssociatedAccount = append(b.AssociatedAccount, associatedAccount)
}

func (b *Base) IsPasswordExist() bool {
	return b.Password != ""
}

type Account struct {
	*redifu.Record
	Base
}

func (asql *Account) GenerateAccessToken(jwtSecret string, jwtTokenIssuer string, jwtTokenLifeSpan time.Duration, sessionID string) (string, error) {
	timeNow := time.Now().UTC()
	expirestAt := timeNow.Add(jwtTokenLifeSpan)

	userClaims := UserClaims{
		UUID:      asql.GetUUID(),
		RandId:    asql.GetRandId(),
		Name:      asql.Name,
		Username:  asql.Username,
		Email:     asql.Email,
		Avatar:    asql.Avatar,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: jwtTokenIssuer,
			IssuedAt: &jwt.NumericDate{
				Time: timeNow,
			},
			ExpiresAt: &jwt.NumericDate{
				Time: expirestAt,
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func New() *Account {
	account := &Account{
		Base: Base{},
	}
	account.EmailVerified = false
	redifu.InitRecord(account)
	return account
}

type AccountReference struct {
	*redifu.Record
	AccountRandId string `json:"accountRandId"`
}

func (ar *AccountReference) SetAccountRandId(accountRandId string) {
	ar.AccountRandId = accountRandId
}

func NewReference() *AccountReference {
	accountReference := &AccountReference{}
	redifu.InitRecord(accountReference)
	return accountReference
}
