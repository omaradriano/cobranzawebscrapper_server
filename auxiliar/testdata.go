package auxiliar

var DbItemsTest = []CobranzaItem{
	{
		Asegurado:    "Omar Adrian Acosta Santiago",
		Contratante:  "Contratante 1",
		DiaCobro:     1,
		Estatus:      "Estatus 1",
		FechaEmision: "FechaEmision 1",
		FormaPago:    "FormaPago 1",
		MedioCobro:   "MedioCobro 1",
		NumPoliza:    "NumPoliza 1",
		Plan:         "Plan 1",
		TipoSeguro:   "TipoSeguro 1",
	},
	{
		Asegurado:    "Asegurado 2",
		Contratante:  "Contratante 2",
		DiaCobro:     2,
		Estatus:      "Estatus 2",
		FechaEmision: "FechaEmision 2",
		FormaPago:    "FormaPago 2",
		MedioCobro:   "MedioCobro 2",
		NumPoliza:    "NumPoliza 2",
		Plan:         "Plan 2",
		TipoSeguro:   "TipoSeguro 2",
	},
	{
		Asegurado:    "Asegurado 3",
		Contratante:  "Contratante 3",
		DiaCobro:     3,
		Estatus:      "Estatus 3",
		FechaEmision: "FechaEmision 3",
		FormaPago:    "FormaPago 3",
		MedioCobro:   "MedioCobro 3",
		NumPoliza:    "NumPoliza 3",
		Plan:         "Plan 3",
		TipoSeguro:   "TipoSeguro 3",
	},
}

type CobranzaItem struct {
	Asegurado    string `json:"asegurado"`
	Contratante  string `json:"contratante"`
	DiaCobro     int8   `json:"diaCobro"`
	Estatus      string `json:"estatus"`
	FechaEmision string `json:"fechaEmision"`
	FormaPago    string `json:"formaPago"`
	MedioCobro   string `json:"medioCobro"`
	NumPoliza    string `json:"numPoliza"`
	Plan         string `json:"plan"`
	TipoSeguro   string `json:"tipoSeguro"`
}
