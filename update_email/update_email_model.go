package update_email

import (
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"time"
)

type UpdateEmail struct {
	*redifu.Record       `bson:",inline" json:",inline"`
	AccountUUID          string    `db:"account_uuid"`
	PreviousEmailAddress string    `db:"previous_email_address"`
	NewEmailAddress      string    `db:"new_email_address"`
	Token                string    `db:"token"`
	ExpiredAt            time.Time `db:"expired_at"`
}

func (ue *UpdateEmail) SetAccountUUID(account *account.Account) {
	ue.AccountUUID = account.UUID
}

func (ue *UpdateEmail) SetPreviousEmailAddress(email string) {
	ue.PreviousEmailAddress = email
}

func (ue *UpdateEmail) SetNewEmailAddress(email string) {
	ue.NewEmailAddress = email
}

func (ue *UpdateEmail) SetToken() {
	token := item.RandId()
	ue.Token = token
}

func (ue *UpdateEmail) SetExpiration() {
	ue.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (ue *UpdateEmail) Validate(token string, bypassExpiration bool) error {
	if !bypassExpiration {
		time := time.Now().UTC()
		if time.After(ue.ExpiredAt) {
			return shared.RequestExpired
		}
	}

	if ue.Token != token {
		return shared.InvalidToken
	}
	return nil
}

func NewUpdateEmailRequestSQL() UpdateEmail {
	ue := UpdateEmail{}
	redifu.InitRecord(&ue)
	return ue
}
