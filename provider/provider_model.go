package provider

import (
	"errors"
	"github.com/21strive/redifu"
)

var ProviderNotFound = errors.New("provider not found")

type Provider struct {
	*redifu.Record
	Name     string `json:"name,omitempty" db:"-"`
	Email    string `json:"email,omitempty" db:"-"`
	Uuid     string `json:"uuid,omitempty" db:"-"`
	Sub      string `json:"sub,omitempty" db:"-"`
	Provider string `json:"provider,omitempty" db:"-"`
}

func NewProvider() *Provider {
	provider := &Provider{}
	redifu.InitRecord(provider)
	return provider
}
