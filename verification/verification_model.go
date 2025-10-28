package verification

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/21strive/commonuser/account"
	"github.com/21strive/redifu"
)

type Verification struct {
	*redifu.Record
	AccountUUID string `db:"accountuuid"`
	Code        string `db:"code"`
}

func (v *Verification) SetAccount(account *account.Account) {
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
	return hex.EncodeToString(hash[:]) == v.Code
}

func New() *Verification {
	verification := &Verification{}
	redifu.InitRecord(verification)
	return verification
}
