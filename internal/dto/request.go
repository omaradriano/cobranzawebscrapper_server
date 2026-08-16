package dto

type PostItem_Poliza struct {
	Asegurado     string      `json:"asegurado"`
	Contratante   string      `json:"contratante"`
	DiaCobro      int16       `json:"dia_cobro"`
	Direccion     Address     `json:"direccion"`
	Estatus       string      `json:"estatus"`
	FechaEmision  string      `json:"fecha_emision"`
	FormaPago     string      `json:"forma_pago"`
	MedioCobro    string      `json:"medio_cobro"`
	NumPoliza     string      `json:"num_poliza"`
	Plan          string      `json:"plan"`
	TipoSeguro    string      `json:"tipo_seguro"`
	Asegurados    []Asegurado `json:"asegurados"`
	Telefono      string      `json:"telefono"`
	SumaAsegurada string      `json:"suma_asegurada"`
	Pais          string      `json:"pais"`
	Email         string      `json:"email"`
	Moneda        string      `json:"moneda"`
	UltimoPago    string      `json:"ultimo_pago"`
}

type PostItems_Poliza struct {
	Payload []PostItem_Poliza `json:"payload"`
}

type PatchItem_Poliza struct {
	NumPoliza string `json:"numpoliza"`
	DiaCobro  int16  `json:"dia_cobro"`
}

type CobranzaItemPayment struct {
	Poliza     string `json:"poliza"`
	PaidPeriod string `json:"paid_period"`
	Agente     string `json:"agente"`
}

type GetItem_Poliza_Filters struct {
	Agente_id  int               `json:"agente_id,omitempty"`
	Filters    map[string]string `json:"filters"`
	PageSize   int               `json:"pageSize"`
	CurentPage int               `json:"currentPage"`
}

type Token struct {
	Token string `json:"token"`
}

type Google_Token struct {
	Payload Token
}

type SetPasswordCredentials struct {
	ResetToken   string `json:"resettoken"`
	Password     string `json:"password"`
	NumeroAsesor string `json:"no_asesor,omitempty"`
	Aseguradora  string `json:"insurance,omitempty"`
}

type UserAseguradorRegister struct {
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	Insurance    *string `json:"insurance,omitempty"`
	NumeroAsesor string  `json:"no_asesor"`
}

type LoginUserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResetPasswordCredentials struct {
	Email string `json:"email"`
}
