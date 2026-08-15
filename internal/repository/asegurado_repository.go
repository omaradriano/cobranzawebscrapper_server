package repository

import (
	"context"

	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"gorm.io/gorm"
)

type AseguradoRepository interface {
	FindByPolizaID(ctx context.Context, polizaID int) ([]models.Asegurado, error)
	Create(ctx context.Context, asegurado *models.Asegurado) error
	BulkCreate(ctx context.Context, asegurados []models.Asegurado) error
}

type aseguradoRepository struct {
	db *gorm.DB
}

func NewAseguradoRepository(db *gorm.DB) AseguradoRepository {
	return &aseguradoRepository{db: db}
}

func (r *aseguradoRepository) FindByPolizaID(ctx context.Context, polizaID int) ([]models.Asegurado, error) {
	var asegurados []models.Asegurado
	err := r.db.WithContext(ctx).Where("poliza_id = ?", polizaID).Find(&asegurados).Error
	return asegurados, err
}

func (r *aseguradoRepository) Create(ctx context.Context, asegurado *models.Asegurado) error {
	return r.db.WithContext(ctx).Create(asegurado).Error
}

func (r *aseguradoRepository) BulkCreate(ctx context.Context, asegurados []models.Asegurado) error {
	if len(asegurados) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&asegurados).Error
}
