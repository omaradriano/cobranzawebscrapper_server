package db

import (
	"context"
	"fmt"
	"time"

	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var GormDB *gorm.DB

type TokenValidator interface {
	ValidateResetToken(token string) (uuid, email, noAgente string, expires time.Time, err error)
	ValidateConfirmationToken(token string) (uuid string, expires time.Time, err error)
}

var AgenteRepo TokenValidator

type tokenValidatorAdapter struct {
	repo interface {
		ValidateResetToken(ctx context.Context, token string) (string, string, string, time.Time, error)
		ValidateConfirmationToken(ctx context.Context, token string) (string, time.Time, error)
	}
}

func (a *tokenValidatorAdapter) ValidateResetToken(token string) (string, string, string, time.Time, error) {
	return a.repo.ValidateResetToken(context.Background(), token)
}

func (a *tokenValidatorAdapter) ValidateConfirmationToken(token string) (string, time.Time, error) {
	return a.repo.ValidateConfirmationToken(context.Background(), token)
}

func SetTokenValidator(repo interface {
	ValidateResetToken(ctx context.Context, token string) (string, string, string, time.Time, error)
	ValidateConfirmationToken(ctx context.Context, token string) (string, time.Time, error)
}) {
	AgenteRepo = &tokenValidatorAdapter{repo: repo}
}

func CreateGormConn() (*gorm.DB, error) {
	connStr := env.Envs.DB_URL

	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL con GORM: %w", err)
	}

	return db, nil
}
