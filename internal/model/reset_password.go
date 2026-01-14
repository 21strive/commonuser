package model

import (
	"github.com/21strive/commonuser/internal/constant"
	"github.com/21strive/item"
	"github.com/21strive/redifu"
	"time"
)

type ResetPassword struct {
	*redifu.Record `bson:",inline" json:",inline"`
	AccountUUID    string    `db:"accountuuid"`
	Token          string    `db:"token"`
	ExpiredAt      time.Time `db:"expiredat"`
}

func (rpsql *ResetPassword) SetAccount(account *Account) {
	rpsql.AccountUUID = account.GetUUID()
}

func (rpsql *ResetPassword) SetToken() {
	rpsql.Token = item.RandId()
}

func (rpsql *ResetPassword) SetExpiredAt(expirationTime *time.Time) {
	rpsql.ExpiredAt = *expirationTime
}

func (rpsql *ResetPassword) IsExpired() bool {
	return time.Now().UTC().After(rpsql.ExpiredAt)
}

func (rpsql *ResetPassword) Validate(token string) error {
	time := time.Now().UTC()
	if time.After(rpsql.ExpiredAt) {
		return constant.RequestExpired
	}
	if rpsql.Token != token {
		return constant.InvalidToken
	}
	return nil
}

func New() *ResetPassword {
	request := &ResetPassword{}
	redifu.InitRecord(request)
	request.ExpiredAt = time.Now().Add(time.Hour * 48)
	return request
}
