package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Poliza struct {
	PolizaID         int        `gorm:"primaryKey;column:poliza_id" json:"-"`
	PolizaUUID       uuid.UUID  `gorm:"column:poliza_uuid;type:uuid;default:gen_random_uuid();uniqueIndex" json:"poliza_uuid"`
	DiaCobro         int16      `gorm:"column:dia_cobro;not null" json:"dia_cobro"`
	Estatus          string     `gorm:"column:estatus;size:20;not null" json:"estatus"`
	FechaEmision     time.Time  `gorm:"column:fecha_emision;not null" json:"fecha_emision"`
	FormaPago        string     `gorm:"column:forma_pago;size:20;not null" json:"forma_pago"`
	MedioCobro       string     `gorm:"column:medio_cobro;size:50;not null" json:"medio_cobro"`
	NumPoliza        string     `gorm:"column:numpoliza;size:20;not null;uniqueIndex" json:"numpoliza"`
	Plan             string     `gorm:"column:plan;size:100;not null" json:"plan"`
	TipoSeguro       string     `gorm:"column:tipo_seguro;size:50;not null" json:"tipo_seguro"`
	AddrCalle        string     `gorm:"column:addr_calle;size:200;default:'No definido'" json:"addr_calle"`
	AddrCodigoPostal string     `gorm:"column:addr_codigopostal;size:10;default:'00000'" json:"addr_codigopostal"`
	AddrCiudad       string     `gorm:"column:addr_ciudad;size:100;default:'No definido'" json:"addr_ciudad"`
	AddrColonia      string     `gorm:"column:addr_colonia;size:100;default:'No definido'" json:"addr_colonia"`
	AddrEstado       string     `gorm:"column:addr_estado;size:100;default:'No definido'" json:"addr_estado"`
	LastModified     *time.Time `gorm:"column:last_modified;default:now()" json:"last_modified"`
	Moneda           *string    `gorm:"column:moneda;size:20" json:"moneda"`
	Telefono         *string    `gorm:"column:telefono;size:15" json:"telefono"`
	SumaAsegurada    *string    `gorm:"column:suma_asegurada;size:20" json:"suma_asegurada"`
	Email            *string    `gorm:"column:email;size:50" json:"email"`
	Pais             *string    `gorm:"column:pais;size:50" json:"pais"`
	AgenteID         *int64     `gorm:"column:agente_id" json:"-"`

	Agente      *Agente      `gorm:"foreignKey:AgenteID" json:"agente,omitempty"`
	Asegurados  []Asegurado  `gorm:"foreignKey:PolizaID" json:"asegurados,omitempty"`
	PaymentConf *PaymentConf `gorm:"foreignKey:PolizaID" json:"payment_conf,omitempty"`
	PaymentLogs []PaymentLog `gorm:"foreignKey:PolizaID" json:"payment_logs,omitempty"`
}

func (Poliza) TableName() string { return "polizas" }

func (p *Poliza) BeforeCreate(tx *gorm.DB) error {
	if p.PolizaUUID == uuid.Nil {
		p.PolizaUUID = uuid.New()
	}
	return nil
}
