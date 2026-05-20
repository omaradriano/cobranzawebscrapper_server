package internal

import "database/sql"

type Env struct {
	DB *sql.DB
}

type CtxKey struct{}

/*
DTO para subir datos a la base de datos
*/
type PostItem_Poliza struct {
	Asegurado     string      `json:"asegurado"`
	Contratante   string      `json:"contratante"`
	DiaCobro      int8        `json:"dia_cobro"`
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
}

type Asegurado struct {
	Nombre      string `json:"nombre"`
	IsPrincipal bool   `json:"is_principal"`
	Cumpleanos  string `json:"birthday"`
}

type Address struct {
	Calle        string `json:"calle"`
	CodigoPostal string `json:"codigo_postal"`
	Ciudad       string `json:"ciudad"`
	Estado       string `json:"estado"`
	Colonia      string `json:"colonia"`
}

/*
DTO para obtener datos de la base de datos
*/
type GetItem_Poliza struct {
	DiaCobro           int8        `json:"diaCobro"`
	Estatus            string      `json:"estatus"`
	FechaEmision       string      `json:"fecha_emision"`
	FormaPago          string      `json:"forma_pago"`
	MedioCobro         string      `json:"medio_cobro"`
	NumPoliza          string      `json:"num_poliza"`
	Plan               string      `json:"plan"`
	TipoSeguro         string      `json:"tipo_seguro"`
	Moneda             string      `json:"moneda"`
	Pais               string      `json:"pais"`
	Asegurados         []Asegurado `json:"asegurados"`
	Email              string      `json:"email"`
	Telefono           string      `json:"telefono"`
	SumaAsegurada      string      `json:"suma_asegurada"`
	UltimaModificacion string      `json:"last_modified"`

	Direccion Address `json:"direccion"`

	SiguientePago string `json:"next_payment"`
	PolizaUUID    string `json:"poliza_uuid"`
	AgenteUUID    string `json:"agente_uuid"`
}

type GetItem_Poliza_Filters struct {
	Agente_id  int               `json:"agente_id,omitempty"`
	Filters    map[string]string `json:"filters"`
	PageSize   int               `json:"pageSize"`
	CurentPage int               `json:"currentPage"`
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

type PostItems_Poliza struct {
	Payload []PostItem_Poliza `json:"payload"`
}

type CobranzaItems struct {
	Payload []GetItem_Poliza `json:"payload"`
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

/*
 * AUTHENTICATION
 */

type Token struct {
	Token string `json:"token"`
}

type Google_Token struct {
	Payload Token
}

type JWTClaims struct {
	Email         string `json:"email"`
	AgenteUUID    string `json:"agente_uuid"`
	NoAgente      string `json:"no_agente"`
	Role          string `json:"agente_role"`
	InsuranceName string `json:"insurance_name"`
	InsuranceID   string `json:"insurance_id"`
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
type DbUserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ResetPasswordCredentials struct {
	Email string `json:"email"`
}

type PolizasUserDetails struct {
	Total             int `json:"total"`
	Activas           int `json:"activas"`
	Inactivas         int `json:"inactivas"`
	PorVencer         int `json:"por_vencer"`
	CoberturaActiva   int `json:"cobertura_activa"`
	SinPagoRegistrado int `json:"sin_pago_registrado"`
}
