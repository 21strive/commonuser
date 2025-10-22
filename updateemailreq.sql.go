package commonuser

import (
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"time"
)

type UpdateEmailRequestSQL struct {
	*redifu.SQLItem      `bson:",inline" json:",inline"`
	AccountUUID          string    `db:"account_uuid"`
	PreviousEmailAddress string    `db:"previous_email_address"`
	NewEmailAddress      string    `db:"new_email_address"`
	UpdateToken          string    `db:"update_token"`
	ExpiredAt            time.Time `db:"expired_at"`
}

func (ue *UpdateEmailRequestSQL) SetAccountUUID(account *Account) {
	ue.AccountUUID = account.UUID
}

func (ue *UpdateEmailRequestSQL) SetPreviousEmailAddress(email string) {
	ue.PreviousEmailAddress = email
}

func (ue *UpdateEmailRequestSQL) SetNewEmailAddress(email string) {
	ue.NewEmailAddress = email
}

func (ue *UpdateEmailRequestSQL) SetResetToken() {
	token := item.RandId()
	ue.UpdateToken = token
}

func (ue *UpdateEmailRequestSQL) SetExpiration() {
	ue.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (ue *UpdateEmailRequestSQL) Validate(updateToken string) error {
	time := time.Now().UTC()
	if time.After(ue.ExpiredAt) {
		return RequestExpired
	}

	if ue.UpdateToken != updateToken {
		return InvalidToken
	}
	return nil
}

func NewUpdateEmailRequestSQL() UpdateEmailRequestSQL {
	ue := UpdateEmailRequestSQL{}
	redifu.InitSQLItem(&ue)
	return ue
}
