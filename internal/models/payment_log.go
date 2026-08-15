package models

import "time"

type PaymentLog struct {
	PaymentLogID int        `gorm:"primaryKey;column:payment_log_id" json:"-"`
	LastUpdated  *time.Time `gorm:"column:last_updated;default:now()" json:"last_updated"`
	PaidPeriod   time.Time  `gorm:"column:paid_period;not null" json:"paid_period"`
	PolizaID     *int64     `gorm:"column:poliza_id" json:"-"`
	AgenteID     *int64     `gorm:"column:agente_id" json:"-"`

	Poliza *Poliza `gorm:"foreignKey:PolizaID" json:"poliza,omitempty"`
	Agente *Agente `gorm:"foreignKey:AgenteID" json:"agente,omitempty"`
}

func (PaymentLog) TableName() string { return "polizas_payments_log" }
