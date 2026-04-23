package internal

import "database/sql"

type Env struct {
	DB *sql.DB
}

type CtxKey struct{}

type CobranzaItem struct {
	Asegurado    string `json:"asegurado"`
	Contratante  string `json:"contratante"`
	DiaCobro     int8   `json:"dia_cobro"`
	Estatus      string `json:"estatus"`
	FechaEmision string `json:"fecha_emision"`
	FormaPago    string `json:"forma_pago"`
	MedioCobro   string `json:"medio_cobro"`
	NumPoliza    string `json:"num_poliza"`
	Plan         string `json:"plan"`
	TipoSeguro   string `json:"tipo_seguro"`
	Asegurador   string `json:"contratante_uuid,omitempty"`

	Direccion Address `json:"direccion"`
}

type PolizaDataItem struct {
	Asegurado    string `json:"asegurado"`
	Contratante  string `json:"contratante"`
	DiaCobro     int8   `json:"diaCobro"`
	Estatus      string `json:"estatus"`
	FechaEmision string `json:"fecha_emision"`
	FormaPago    string `json:"forma_pago"`
	MedioCobro   string `json:"medio_cobro"`
	NumPoliza    string `json:"num_poliza"`
	Plan         string `json:"plan"`
	TipoSeguro   string `json:"tipo_seguro"`
	HasLog       int    `json:"haslog"`

	Direccion Address `json:"direccion"`

	SiguientePago string `json:"next_payment"`
	PolizaUUID    string `json:"poliza_uuid"`
	UserUUID      string `json:"user_uuid"`
}

type Address struct {
	Calle        string `json:"calle"`
	CodigoPostal string `json:"codigo_postal"`
	Ciudad       string `json:"ciudad"`
	Estado       string `json:"estado"`
	Colonia      string `json:"colonia"`
}

type HttpError struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HttpSuccess struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type CobranzaItems struct {
	Payload []CobranzaItem `json:"payload"`
}

type CobranzaPatchItems struct {
	Payload []CobranzaPatchItem `json:"payload"`
}

type CobranzaPatchItem struct {
	ContratanteID  int    `json:"contratanteID"`
	NumPoliza      string `json:"numpoliza"`
	Notificaciones bool   `json:"notificaciones"`
}

type CobranzaItemPayment struct {
	Poliza     string `json:"poliza"`
	PaidPeriod string `json:"paid_period"`
	Asegurador string `json:"asegurador"`
}

type Token struct {
	Token string `json:"token"`
}

type Google_Token struct {
	Payload Token
}

type Google_User_Response struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Google_ID string `json:"sub"`
}

type Server_Response_With_Token struct {
	JWT_Token string `json:"jwt_token"`
	Email     string `json:"email,omitempty"`
}

type Verify_Password_Response struct {
	HasPassword   bool   `json:"haspassword"`
	PasswordToken string `json:"passtoken,omitempty"`
}

type SetPasswordCredentials struct {
	ResetToken string `json:"resettoken"`
	Password   string `json:"password"`
}

type UserAseguradorRegister struct {
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	Insurance *string `json:"insurance,omitempty"`
}

type LoginUserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type DbUserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResetPasswordCredentials struct {
	Email string `json:"email"`
}
