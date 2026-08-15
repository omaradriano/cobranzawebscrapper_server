package models

type AseguradoraConf struct {
	AseguradoraID int    `gorm:"primaryKey;column:aseguradora_id" json:"aseguradora_id"`
	Nombre        string `gorm:"column:nombre;size:200;not null" json:"nombre"`
	Clave         string `gorm:"column:clave;size:5;not null;uniqueIndex" json:"clave"`

	Agentes []Agente `gorm:"foreignKey:AseguradoraID" json:"agentes,omitempty"`
}

func (AseguradoraConf) TableName() string { return "aseguradoras_conf" }
