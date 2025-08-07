package lib

import (
	"github.com/21strive/item"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matthewhartstonge/argon2"
	"time"
)

type UserClaims struct {
	UUID              string    `json:"uuid"` // user uuid
	Name              string    `json:"name"`
	Username          string    `json:"username,omitempty"`
	Email             string    `json:"email,omitempty"`
	Avatar            string    `json:"avatar,omitempty"`
	PasswordUpdatedAt time.Time `json:"passwordupdatedat,omitempty"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	UUID string `json:"uuid"` // user uuid
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
	PasswordUpdatedAt time.Time           `json:"-" db:"passwordupdatedat"`
	Email             string              `json:"email,omitempty" db:"email"`
	Avatar            string              `json:"avatar,omitempty" db:"avatar"`
	AssociatedAccount []AssociatedAccount `json:"associatedAccount,omitempty" db:"-"`
	Suspended         bool                `json:"suspended,omitempty" db:"suspended"`
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
	b.PasswordUpdatedAt = time.Now().UTC()
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

func (b *Base) SetAvatar(avatar string) {
	b.Avatar = avatar
}

func (b *Base) SetAssociatedAccount(associatedAccount AssociatedAccount) {
	b.AssociatedAccount = append(b.AssociatedAccount, associatedAccount)
}

func (b *Base) Suspend() {
	b.Suspended = true
}

func (b *Base) Release() {
	b.Suspended = false
}

func (b *Base) IsSuspended() bool {
	return b.Suspended
}

func (b *Base) IsPasswordExist() bool {
	return b.Password != ""
}

type Account struct {
	*item.Foundation
	Base
}
