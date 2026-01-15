package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/21strive/redifu"
)

var VerificationNotFound = errors.New("Verification not found")
var InvalidVerificationCode = errors.New("Invalid verification code")

type Verification struct {
	*redifu.Record
	AccountUUID string `db:"accountuuid"`
	Code        string `db:"code"`
}

func (v *Verification) SetAccount(account *Account) {
	v.AccountUUID = account.GetUUID()
}

func (v *Verification) SetCode() string {
	numericString := ""
	for i := int64(0); i < 4; i++ {
		digit, _ := rand.Int(rand.Reader, big.NewInt(10))
		numericString += digit.String()
	}

	hash := sha256.Sum256([]byte(numericString))
	v.Code = hex.EncodeToString(hash[:])

	return numericString
}

func (v *Verification) Validate(code string) bool {
	hash := sha256.Sum256([]byte(code))
	stringifiedHash := hex.EncodeToString(hash[:])
	return stringifiedHash == v.Code
}

func NewVerification() *Verification {
	verification := &Verification{}
	redifu.InitRecord(verification)
	return verification
}
