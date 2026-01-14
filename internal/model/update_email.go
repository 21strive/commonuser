package model

import (
	"github.com/21strive/commonuser/internal/constant"
	"github.com/21strive/commonuser/update_email"
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"github.com/matthewhartstonge/argon2"
	"time"
)

type UpdateEmail struct {
	*redifu.Record       `bson:",inline" json:",inline"`
	AccountUUID          string    `db:"account_uuid"`
	PreviousEmailAddress string    `db:"previous_email_address"`
	NewEmailAddress      string    `db:"new_email_address"`
	Token                string    `db:"token"`
	RevokeToken          string    `db:"revoke_token"`
	Processed            bool      `db:"processed"`
	ExpiredAt            time.Time `db:"expired_at"`
}

func (ue *UpdateEmail) SetAccount(account *Account) {
	ue.AccountUUID = account.UUID
}

func (ue *UpdateEmail) SetPreviousEmailAddress(email string) {
	ue.PreviousEmailAddress = email
}

func (ue *UpdateEmail) SetNewEmailAddress(email string) {
	ue.NewEmailAddress = email
}

func (ue *UpdateEmail) SetToken() (string, error) {
	token := item.RandId()
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(token))
	if err != nil {
		return token, err
	}

	ue.Token = string(encoded)
	return token, nil
}

func (ue *UpdateEmail) SetRevokeToken() (string, error) {
	token := item.RandId()
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(token))
	if err != nil {
		return token, err
	}
	ue.RevokeToken = string(encoded)
	return token, nil
}

func (ue *UpdateEmail) SetExpiration() {
	ue.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (ue *UpdateEmail) SetProcessed() {
	ue.Processed = true
}

func (ue *UpdateEmail) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(ue.ExpiredAt) {
		return constant.RequestExpired
	}

	match, err := argon2.VerifyEncoded([]byte(token), []byte(ue.Token))
	if err != nil {
		return err
	}
	if !match {
		return update_email.InvalidToken
	}

	return nil
}

func (ue *UpdateEmail) ValidateRevoke(token string) error {
	match, err := argon2.VerifyEncoded([]byte(token), []byte(ue.RevokeToken))
	if err != nil {
		return err
	}
	if !match {
		return update_email.InvalidToken
	}

	return nil
}

func (ue *UpdateEmail) IsExpired() bool {
	return time.Now().UTC().After(ue.ExpiredAt)
}

func New() *UpdateEmail {
	ue := &UpdateEmail{}
	redifu.InitRecord(ue)

	ue.SetExpiration()
	ue.Processed = false
	return ue
}
