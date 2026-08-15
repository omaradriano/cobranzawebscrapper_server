package repository

import (
	"context"

	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	CreateLog(ctx context.Context, log *models.PaymentLog) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateLog(ctx context.Context, log *models.PaymentLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}
