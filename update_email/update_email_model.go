package update_email

import (
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/shared"
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
	Revoked              bool      `db:"revoked"`
	ExpiredAt            time.Time `db:"expired_at"`
}

func (ue *UpdateEmail) SetAccount(account *account.Account) {
	ue.AccountUUID = account.UUID
}

func (ue *UpdateEmail) SetPreviousEmailAddress(email string) {
	ue.PreviousEmailAddress = email
}

func (ue *UpdateEmail) SetNewEmailAddress(email string) {
	ue.NewEmailAddress = email
}

func (ue *UpdateEmail) SetToken() error {
	token := item.RandId()
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(token))
	if err != nil {
		return err
	}

	ue.Token = string(encoded)
	return nil
}

func (ue *UpdateEmail) SetRevokeToken() error {
	token := item.RandId()
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(token))
	if err != nil {
		return err
	}
	ue.RevokeToken = string(encoded)
	return nil
}

func (ue *UpdateEmail) SetExpiration() {
	ue.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (ue *UpdateEmail) SetProcessed() {
	ue.Processed = true
}

func (ue *UpdateEmail) SetRevoked() {
	ue.Revoked = true
}

func (ue *UpdateEmail) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(ue.ExpiredAt) {
		return shared.RequestExpired
	}

	match, err := argon2.VerifyEncoded([]byte(token), []byte(ue.Token))
	if err != nil {
		return err
	}
	if !match {
		return InvalidToken
	}

	return nil
}

func (ue *UpdateEmail) ValidateRevoke(token string) error {
	if ue.RevokeToken != token {
		return shared.InvalidToken
	}

	match, err := argon2.VerifyEncoded([]byte(token), []byte(ue.RevokeToken))
	if err != nil {
		return err
	}
	if !match {
		return InvalidToken
	}

	return nil
}

func (ue *UpdateEmail) IsExpired() bool {
	return time.Now().UTC().After(ue.ExpiredAt)
}

func New() *UpdateEmail {
	ue := &UpdateEmail{}
	redifu.InitRecord(ue)

	ue.SetToken()
	ue.SetRevokeToken()
	ue.SetExpiration()
	ue.Processed = false
	return ue
}
