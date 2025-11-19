package provider

import (
	"errors"
	"github.com/21strive/commonuser/account"
	"github.com/21strive/redifu"
)

var ProviderNotFound = errors.New("provider not found")

type Provider struct {
	*redifu.Record
	Name        string `json:"name"`
	Email       string `json:"email"`
	Sub         string `json:"sub"`
	Issuer      string `json:"issuer"`
	AccountUUID string `json:"-"`
}

func (p *Provider) SetName(name string) {
	p.Name = name
}

func (p *Provider) SetEmail(email string) {
	p.Email = email
}

func (p *Provider) SetSub(sub string) {
	p.Sub = sub
}

func (p *Provider) SetIssuer(issuer string) {
	p.Issuer = issuer
}

func (p *Provider) SetAccount(account *account.Account) {
	p.AccountUUID = account.UUID
}

func New() *Provider {
	provider := &Provider{}
	redifu.InitRecord(provider)
	return provider
}
