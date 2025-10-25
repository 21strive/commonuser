package reset_password

import (
	"github.com/21strive/commonuser/account"
	"github.com/21strive/commonuser/shared"
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"time"
)

type ResetPassword struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	AccountUUID     string    `db:"accountuuid"`
	Token           string    `db:"token"`
	ExpiredAt       time.Time `db:"expiredat"`
}

func (rpsql *ResetPassword) SetAccountUUID(account *account.Account) {
	rpsql.AccountUUID = account.GetUUID()
}

func (rpsql *ResetPassword) SetToken() {
	rpsql.Token = item.RandId()
}

func (rpsql *ResetPassword) SetExpiredAt() {
	rpsql.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (rpsql *ResetPassword) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(rpsql.ExpiredAt) {
		return shared.RequestExpired
	}
	if rpsql.Token != token {
		return shared.InvalidToken
	}
	return nil
}

func NewResetPasswordSQL() ResetPassword {
	request := ResetPassword{}
	redifu.InitSQLItem(&request)
	return request
}
