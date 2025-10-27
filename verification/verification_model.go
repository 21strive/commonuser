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
	AccountUUID      string `db:"accountuuid"`
	VerificationHash string `db:"verification_hash"`
}

func (v *Verification) SetAccountUUID(account *account.Account) {
	v.AccountUUID = account.GetUUID()
}

func (v *Verification) SetVerification(length int64) string {
	numericString := ""
	for i := int64(0); i < length; i++ {
		digit, _ := rand.Int(rand.Reader, big.NewInt(10))
		numericString += digit.String()
	}

	hash := sha256.Sum256([]byte(numericString))
	v.VerificationHash = hex.EncodeToString(hash[:])

	return numericString
}

func (v *Verification) Validate(verification string) bool {
	hash := sha256.Sum256([]byte(verification))
	return hex.EncodeToString(hash[:]) == v.VerificationHash
}

func NewVerification() *Verification {
	verification := &Verification{}
	redifu.InitRecord(verification)
	return verification
}
