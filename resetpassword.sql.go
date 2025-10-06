package commonuser

import (
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"time"
)

type ResetPasswordRequestSQL struct {
	*redifu.SQLItem `bson:",inline" json:",inline"`
	AccountUUID     string    `db:"accountuuid"`
	Token           string    `db:"token"`
	ExpiredAt       time.Time `db:"expiredat"`
}

func (rpsql *ResetPasswordRequestSQL) SetAccountUUID(account *Account) {
	rpsql.AccountUUID = account.GetUUID()
}

func (rpsql *ResetPasswordRequestSQL) SetToken() {
	rpsql.Token = item.RandId()
}

func (rpsql *ResetPasswordRequestSQL) SetExpiredAt() {
	rpsql.ExpiredAt = time.Now().Add(time.Hour * 48)
}

func (rpsql *ResetPasswordRequestSQL) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(rpsql.ExpiredAt) {
		return RequestExpired
	}
	if rpsql.Token != token {
		return InvalidToken
	}
	return nil
}

func NewResetPasswordSQL() ResetPasswordRequestSQL {
	request := ResetPasswordRequestSQL{}
	redifu.InitSQLItem(&request)
	return request
}
