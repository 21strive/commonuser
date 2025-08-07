package lib

import "github.com/21strive/redifu"

type AccountMongo struct {
	*redifu.MongoItem `bson:",inline" json:",inline"`
	*Base             `bson:",inline" json:",inline"`
}
