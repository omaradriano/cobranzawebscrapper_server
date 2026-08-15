package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Asegurado struct {
	AseguradoID    int        `gorm:"primaryKey;column:asegurado_id" json:"-"`
	AseguradoUUID  uuid.UUID  `gorm:"column:asegurado_uuid;type:uuid;default:gen_random_uuid();not null" json:"asegurado_uuid"`
	Birthday       *time.Time `gorm:"column:birthday" json:"birthday"`
	NombreCompleto string     `gorm:"column:nombre_completo;size:200;not null" json:"nombre_completo"`
	IsPrincipal    *bool      `gorm:"column:is_principal" json:"is_principal"`
	PolizaID       *int64     `gorm:"column:poliza_id" json:"-"`

	Poliza *Poliza `gorm:"foreignKey:PolizaID" json:"poliza,omitempty"`
}

func (Asegurado) TableName() string { return "asegurados" }

func (a *Asegurado) BeforeCreate(tx *gorm.DB) error {
	if a.AseguradoUUID == uuid.Nil {
		a.AseguradoUUID = uuid.New()
	}
	return nil
}
