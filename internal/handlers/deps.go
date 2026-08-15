package handlers

import (
	"github.com/omaradriano/cobranzawebscrapper_server/internal/repository"
	"gorm.io/gorm"
)

type Deps struct {
	DB            *gorm.DB
	AgenteRepo    repository.AgenteRepository
	PolizaRepo    repository.PolizaRepository
	AseguradoRepo repository.AseguradoRepository
	PaymentRepo   repository.PaymentRepository
}

var deps *Deps

func InitDeps(gormDB *gorm.DB) {
	deps = &Deps{
		DB:            gormDB,
		AgenteRepo:    repository.NewAgenteRepository(gormDB),
		PolizaRepo:    repository.NewPolizaRepository(gormDB),
		AseguradoRepo: repository.NewAseguradoRepository(gormDB),
		PaymentRepo:   repository.NewPaymentRepository(gormDB),
	}
}
