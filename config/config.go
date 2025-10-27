package config

import "time"

type App struct {
	RecordAge     time.Duration
	PaginationAge time.Duration
	EntityName    string
	TokenLifespan time.Duration
	JWTSecret     string
	JWTIssuer     string
	JWTLifespan   time.Duration
}

func (a *App) GetRecordAge() time.Duration {
	return a.RecordAge
}

func (a *App) GetPaginationAge() time.Duration {
	return a.PaginationAge
}

func (a *App) GetEntityName() string {
	return a.EntityName
}

func DefaultConfig(entityName string, jwtSecret string, jwtIssuer string, jwtLifespan time.Duration) *App {
	return &App{
		RecordAge:     time.Hour * 12,
		PaginationAge: time.Hour * 24,
		TokenLifespan: time.Hour * 24,
		EntityName:    entityName,
		JWTSecret:     jwtSecret,
		JWTIssuer:     jwtIssuer,
		JWTLifespan:   jwtLifespan,
	}
}
