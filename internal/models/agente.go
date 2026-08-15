package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Agente struct {
	AgenteID             int        `gorm:"primaryKey;column:agente_id" json:"-"`
	AgenteUUID           uuid.UUID  `gorm:"column:agente_uuid;type:uuid;default:gen_random_uuid()" json:"agente_uuid"`
	Email                string     `gorm:"column:email;uniqueIndex;size:100;not null" json:"email"`
	PasswordHash         *string    `gorm:"column:password_hash;size:100" json:"-"`
	APIKey               *string    `gorm:"column:api_key;size:100" json:"-"`
	DaysUntilAdvice      int16      `gorm:"column:daysuntiladvice;default:5;not null" json:"daysuntiladvice"`
	GoogleID             *string    `gorm:"column:google_id;size:255;uniqueIndex" json:"-"`
	IsVerified           bool       `gorm:"column:is_verified;default:false" json:"is_verified"`
	VerificationToken    *string    `gorm:"column:verification_token" json:"-"`
	VerificationExpires  *time.Time `gorm:"column:verification_expires" json:"-"`
	ResetToken           *string    `gorm:"column:reset_token" json:"-"`
	ResetExpires         *time.Time `gorm:"column:reset_expires" json:"-"`
	NoAgente             string     `gorm:"column:no_agente;uniqueIndex;size:6;not null" json:"no_agente"`
	AseguradoraID        *int64     `gorm:"column:aseguradora_id" json:"aseguradora_id"`
	Role                 string     `gorm:"column:role;size:10;default:'user'" json:"role"`
	IsSubscribed         bool       `gorm:"column:is_subscribed;default:false" json:"is_subscribed"`
	StripeSubscriptionID *string    `gorm:"column:stripe_subscription_id" json:"-"`
	CancelAtPeriodEnd    bool       `gorm:"column:cancel_at_period_end;default:false" json:"cancel_at_period_end"`
	CurrentPeriodEnd     int64      `gorm:"column:current_period_end;default:0" json:"current_period_end"`

	Aseguradora *AseguradoraConf `gorm:"foreignKey:AseguradoraID" json:"aseguradora,omitempty"`
	Polizas     []Poliza         `gorm:"foreignKey:AgenteID" json:"polizas,omitempty"`
}

func (Agente) TableName() string { return "agentes" }

func (a *Agente) BeforeCreate(tx *gorm.DB) error {
	if a.AgenteUUID == uuid.Nil {
		a.AgenteUUID = uuid.New()
	}
	return nil
}
