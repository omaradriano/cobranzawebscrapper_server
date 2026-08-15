package repository

import (
	"context"
	"time"

	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"gorm.io/gorm"
)

type AgenteRepository interface {
	FindByEmail(ctx context.Context, email string) (*models.Agente, error)
	FindByEmailWithInsurance(ctx context.Context, email string) (*models.Agente, *models.AseguradoraConf, error)
	FindByUUID(ctx context.Context, uuid string) (*models.Agente, error)
	FindIDByUUID(ctx context.Context, uuid string) (int, error)
	FindPasswordByEmail(ctx context.Context, email string) (*string, error)
	Create(ctx context.Context, agente *models.Agente) error
	UpdateResetToken(ctx context.Context, email string, token string, expires time.Time) error
	UpdatePassword(ctx context.Context, uuid string, hash string, noAgente *string, aseguradoraID *string) error
	UpdateVerification(ctx context.Context, uuid string) error
	UpdateSubscription(ctx context.Context, fields map[string]any, condition string, conditionVal any) error
	GetSubscriptionStatus(ctx context.Context, uuid string) (*models.Agente, error)
	GetSubscriptionID(ctx context.Context, uuid string) (string, error)
	ValidateResetToken(ctx context.Context, token string) (string, string, string, time.Time, error)
	ValidateConfirmationToken(ctx context.Context, token string) (string, time.Time, error)
	FindIsVerified(ctx context.Context, token string, uuid string) (bool, error)
	FindEmailByEmail(ctx context.Context, email string) (string, error)
}

type agenteRepository struct {
	db *gorm.DB
}

func NewAgenteRepository(db *gorm.DB) AgenteRepository {
	return &agenteRepository{db: db}
}

func (r *agenteRepository) FindByEmail(ctx context.Context, email string) (*models.Agente, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&agente).Error
	if err != nil {
		return nil, err
	}
	return &agente, nil
}

func (r *agenteRepository) FindByEmailWithInsurance(ctx context.Context, email string) (*models.Agente, *models.AseguradoraConf, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).Preload("Aseguradora").Where("email = ?", email).First(&agente).Error
	if err != nil {
		return nil, nil, err
	}
	return &agente, agente.Aseguradora, nil
}

func (r *agenteRepository) FindByUUID(ctx context.Context, uuid string) (*models.Agente, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).Where("agente_uuid = ?", uuid).First(&agente).Error
	if err != nil {
		return nil, err
	}
	return &agente, nil
}

func (r *agenteRepository) FindIDByUUID(ctx context.Context, uuid string) (int, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).Select("agente_id").Where("agente_uuid = ?", uuid).First(&agente).Error
	if err != nil {
		return 0, err
	}
	return agente.AgenteID, nil
}

func (r *agenteRepository) FindPasswordByEmail(ctx context.Context, email string) (*string, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).Select("password_hash").Where("email = ?", email).First(&agente).Error
	if err != nil {
		return nil, err
	}
	return agente.PasswordHash, nil
}

func (r *agenteRepository) Create(ctx context.Context, agente *models.Agente) error {
	return r.db.WithContext(ctx).Create(agente).Error
}

func (r *agenteRepository) UpdateResetToken(ctx context.Context, email string, token string, expires time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Agente{}).
		Where("email = ?", email).
		Updates(map[string]any{
			"reset_token":   token,
			"reset_expires": expires,
		}).Error
}

func (r *agenteRepository) UpdatePassword(ctx context.Context, uuid string, hash string, noAgente *string, aseguradoraID *string) error {
	updates := map[string]any{
		"password_hash": hash,
		"reset_token":   nil,
		"reset_expires": nil,
	}
	if noAgente != nil {
		updates["no_agente"] = *noAgente
	}
	if aseguradoraID != nil {
		updates["aseguradora_id"] = *aseguradoraID
	}
	return r.db.WithContext(ctx).Model(&models.Agente{}).
		Where("agente_uuid = ?", uuid).
		Updates(updates).Error
}

func (r *agenteRepository) UpdateVerification(ctx context.Context, uuid string) error {
	return r.db.WithContext(ctx).Model(&models.Agente{}).
		Where("agente_uuid = ?", uuid).
		Updates(map[string]any{
			"is_verified":          true,
			"verification_expires": nil,
			"verification_token":   nil,
		}).Error
}

func (r *agenteRepository) UpdateSubscription(ctx context.Context, fields map[string]any, condition string, conditionVal any) error {
	return r.db.WithContext(ctx).Model(&models.Agente{}).
		Where(condition, conditionVal).
		Updates(fields).Error
}

func (r *agenteRepository) GetSubscriptionStatus(ctx context.Context, uuid string) (*models.Agente, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("is_subscribed", "cancel_at_period_end", "current_period_end").
		Where("agente_uuid = ?", uuid).
		First(&agente).Error
	if err != nil {
		return nil, err
	}
	return &agente, nil
}

func (r *agenteRepository) GetSubscriptionID(ctx context.Context, uuid string) (string, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("COALESCE(stripe_subscription_id, '') as stripe_subscription_id").
		Where("agente_uuid = ?", uuid).
		First(&agente).Error
	if err != nil {
		return "", err
	}
	if agente.StripeSubscriptionID == nil {
		return "", nil
	}
	return *agente.StripeSubscriptionID, nil
}

func (r *agenteRepository) ValidateResetToken(ctx context.Context, token string) (string, string, string, time.Time, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("agente_uuid", "email", "reset_expires", "no_agente").
		Where("reset_token = ?", token).
		First(&agente).Error
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	var expires time.Time
	if agente.ResetExpires != nil {
		expires = *agente.ResetExpires
	}
	return agente.AgenteUUID.String(), agente.Email, agente.NoAgente, expires, nil
}

func (r *agenteRepository) ValidateConfirmationToken(ctx context.Context, token string) (string, time.Time, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("agente_uuid", "verification_expires").
		Where("verification_token = ?", token).
		First(&agente).Error
	if err != nil {
		return "", time.Time{}, err
	}
	var expires time.Time
	if agente.VerificationExpires != nil {
		expires = *agente.VerificationExpires
	}
	return agente.AgenteUUID.String(), expires, nil
}

func (r *agenteRepository) FindIsVerified(ctx context.Context, token string, uuid string) (bool, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("is_verified").
		Where("verification_token = ? AND agente_uuid = ?", token, uuid).
		First(&agente).Error
	if err != nil {
		return false, err
	}
	return agente.IsVerified, nil
}

func (r *agenteRepository) FindEmailByEmail(ctx context.Context, email string) (string, error) {
	var agente models.Agente
	err := r.db.WithContext(ctx).
		Select("email").
		Where("email = ?", email).
		First(&agente).Error
	if err != nil {
		return "", err
	}
	return agente.Email, nil
}
