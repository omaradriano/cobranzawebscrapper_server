package models

import "time"

type PaymentConf struct {
	PaymentConfID      int        `gorm:"primaryKey;column:payment_conf_id" json:"-"`
	NextPayment        *time.Time `gorm:"column:next_payment" json:"next_payment"`
	AllowNotifications bool       `gorm:"column:allownotifications;default:false" json:"allownotifications"`
	PolizaID           *int64     `gorm:"column:poliza_id" json:"-"`

	Poliza *Poliza `gorm:"foreignKey:PolizaID" json:"poliza,omitempty"`
}

func (PaymentConf) TableName() string { return "polizas_payments_conf" }
