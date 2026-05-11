package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

/**
 * ApiCheckPasswordExist verifica que exista una contraseña con el usuario con el que se quiere iniciar sesion
 * En caso de no existir una contraseña, el front delegará el camino que se toma para establecer una nueva
 * Esto debido a que necesitamos una contraseña en caso de que el usuario quiera usar credenciales manuales para hacer login
 */
func ApiCheckPasswordExist(w http.ResponseWriter, r *http.Request) {
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	fmt.Println("Request from ApiCheckPasswordExist")
	fmt.Printf("----------------------------------------\n")

	/**
	 * Verificacion para saber si el email registrado tiene una contrasena.
	 */
	var verifiedEmailRes internal.Verify_Password_Response
	email := GetField(r, 0)
	rows, err := db.Client.Query(`SELECT password_hash FROM users_aseguradores WHERE email = $1`, email)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido ejecutar el query para obtener si existe pass", w)
		return
	}
	defer rows.Close()

	found := false
	var hasPassword string

	for rows.Next() {
		if err := rows.Scan(&hasPassword); err != nil {
			fmt.Println(err.Error())
			// services.HandleResponseError(http.StatusInternalServerError, "Error al leer datos", w)
			// return
		} else {
			found = true
		}
	}

	/**
	 * Token generado será enviado como respuesta para poder usarlo despues al momento de establecer contraseña.
	 */
	verifiedEmailRes.HasPassword = found
	if !verifiedEmailRes.HasPassword {
		verifiedEmailRes.PasswordToken, err = services.GenerateSecureToken()
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "No se ha podido generar token de contraseña", w)
			return
		}
		token_expires := time.Now().Add(30 * time.Minute)
		db.Client.Exec(`UPDATE users_aseguradores SET reset_token = $1, reset_expires = $2 WHERE email = $3`, verifiedEmailRes.PasswordToken, token_expires, email)
		services.HandleResponseSuccessWithData(verifiedEmailRes, w)
		return
	}

	services.HandleResponseSuccessWithData(verifiedEmailRes, w)
}

func ApiAuthenticateUserByGoogle(w http.ResponseWriter, r *http.Request) {
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	fmt.Println("Request from ApiAuthenticateUser")
	fmt.Printf("----------------------------------------\n")

	var google_token internal.Google_Token

	err := json.NewDecoder(r.Body).Decode(&google_token)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	/**
	 *
		* INICIA SOLICITUD A GOOGLE PARA VERIFICAR AL USUARIO QUE INTENTA LOGGEAR
	*/

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	url := "https://www.googleapis.com/oauth2/v3/userinfo"
	method := "GET"
	payload := strings.NewReader(``)

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error al armar la solicitud para api google", w)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf(`Bearer %s`, google_token.Payload.Token))

	resp, err := client.Do(req)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "Error al realizar la solicitud para api google", w)
		return
	}
	defer resp.Body.Close()

	var google_user_response internal.Google_User_Response
	var user_uuid string

	err = json.NewDecoder(resp.Body).Decode(&google_user_response)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "No existe respuesta de google_user_response", w)
		return
	}

	/**
	 * INICIA OPERACION PARA LOGGEAR O REGISTRAR AL USUARIO EN BASE DE DATOS
	 */

	rows, err := db.Client.Query(
		`SELECT email, user_uuid FROM users_aseguradores WHERE email = $1`,
		google_user_response.Email,
	)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido hacer peticion de datos", w)
		return
	}
	defer rows.Close()

	// Esta variable va a identificar que haya registros en el query realizado
	found := false

	var email string
	for rows.Next() {
		if err := rows.Scan(&email, &user_uuid); err != nil {
			fmt.Println(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error al leer datos", w)
			return
		}
		found = true
	}

	if err := rows.Err(); err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error en iteración de filas", w)
		return
	}

	var ServerResponse internal.Server_Response_With_Token

	ServerResponse.JWT_Token, err = services.GenerateJWT(user_uuid, email)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error al obtener jwt token", w)
	}

	// fmt.Println(google_user_response)
	ServerResponse.Email = google_user_response.Email

	if !found {
		/**
		 * El usuario no existe por lo que hay que proceder a registro del mismo
		 */
		fmt.Println("No se encontró usuario, hay que crear uno nuevo")
		createUserQuery := `INSERT INTO users_aseguradores (email, is_verified, no_asesor) VALUES ($1, true, '000000')`
		_, err := db.Client.Exec(createUserQuery, google_user_response.Email)
		if err != nil {
			fmt.Println("Error al insertar un nuevo usuario con validacion google:", err)
			services.HandleResponseError(409, "No se ha podido crear un nuevo usuario por autenticacion google", w)
			return
		}
		services.HandleResponseSuccessWithData(ServerResponse, w)
	} else {
		/**
		 * El usuario si existe por lo que se genera token de la sesión
		 */
		fmt.Println("Se encontró usuario, hay que generar el token de sesion")

		w.Header().Set("Authorization", fmt.Sprintf(`Bearer %s`, ServerResponse.JWT_Token))
		services.HandleResponseSuccessWithData(ServerResponse, w)
	}
}

func ApiAuthenticateUserByCredentials(w http.ResponseWriter, r *http.Request) {
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	fmt.Println("Request from ApiAuthenticateUserByCredentials")
	fmt.Printf("----------------------------------------\n")

	var login_credentials internal.LoginUserCredentials
	var db_credentials internal.LoginUserCredentials
	var ServerResponse internal.Server_Response_With_Token
	var user_uuid string
	var is_account_verified bool
	err := json.NewDecoder(r.Body).Decode(&login_credentials)
	if err != nil {
		fmt.Println("Error decoding JSON on ApiAuthenticateUserByCredentials:", err)
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiAuthenticateUserByCredentials", w)
		return
	}

	err = db.Client.QueryRow(`
		SELECT email, password_hash, user_uuid, is_verified
		FROM users_aseguradores
		WHERE email = $1
		`, login_credentials.Email).
		Scan(&db_credentials.Email, &db_credentials.Password, &user_uuid, &is_account_verified)
	fmt.Println(db_credentials.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err.Error())
			services.HandleResponseError(http.StatusUnauthorized, "Credenciales incorrectas", w)
			return
		}

		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error al consultar usuario", w)
		return
	}

	if !is_account_verified {
		services.HandleResponseError(http.StatusUnauthorized, "La cuenta no esta verificada", w)
		return
	}

	hashedPassword, _ := services.HashPassword(login_credentials.Password)

	fmt.Println(hashedPassword)
	fmt.Println(db_credentials.Password)
	// fmt.Println(db_credentials.Password)
	areCredentialsValid := services.CheckPassword(db_credentials.Password, login_credentials.Password)

	if !areCredentialsValid {
		// fmt.Println(err.Error())
		services.HandleResponseError(http.StatusUnauthorized, "Credenciales incorrectas", w)
		return
	}

	ServerResponse.JWT_Token, err = services.GenerateJWT(user_uuid, db_credentials.Email)
	// ServerResponse.Email = db_credentials.Email

	services.HandleResponseSuccessWithData(ServerResponse, w)
}

func ApiCheckSession(w http.ResponseWriter, r *http.Request) {
	// 1. Configuramos Headers (CORS)
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	var session_claims internal.JWTClaims

	// 2. Extraemos los claims que el Middleware inyectó en el contexto
	// Nota: Usa las llaves que definiste en tu middleware (ej. configs.UserIDKey)
	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)
	userEmail, _ := r.Context().Value(middlewares.UserEmailKey).(string)

	// Logging para depuración en tu servidor
	fmt.Printf("--- Sesión Verificada ---\n")
	fmt.Printf("User UUID: %s\n", userUUID)
	fmt.Printf("Email:     %s\n", userEmail)
	fmt.Printf("-------------------------\n")

	session_claims.Email = userEmail

	// 3. Si llegamos aquí, el middleware ya validó el JWT.
	// Simplemente respondemos éxito.
	services.HandleResponseSuccessWithData(session_claims, w)
}

func ApiSetPassword(w http.ResponseWriter, r *http.Request) {
	/*
		 * http://127.0.0.1:3006/v1/auth/setpassword?token=1234abcd
			* body:
			* {
			* 	"password":string,
			* 	"no_asesor":string
			* }
	*/
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	fmt.Println("Request from ApiSetPassword")
	fmt.Printf("----------------------------------------\n")

	var password_credentials internal.SetPasswordCredentials
	var no_asesor string

	password_credentials.ResetToken = r.URL.Query().Get("token")

	err := json.NewDecoder(r.Body).Decode(&password_credentials)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiSetPassword", w)
		return
	}

	fmt.Println(password_credentials.NumeroAsesor)

	user_uuid, email, err := services.ValidateResetToken(password_credentials.ResetToken)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	err = db.Client.QueryRow(`SELECT no_asesor FROM users_aseguradores WHERE user_uuid = $1`, user_uuid).
		Scan(&no_asesor)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	hashed_password, _ := services.HashPassword(password_credentials.Password)

	fmt.Println(password_credentials.NumeroAsesor)

	if no_asesor == "000000" {
		_, err = db.Client.Exec(`
			UPDATE users_aseguradores
			SET password_hash = $1,
				reset_token = NULL,
				reset_expires = NULL,
				no_asesor = $2
			WHERE user_uuid = $3`, hashed_password, password_credentials.NumeroAsesor, user_uuid)
	} else {
		_, err = db.Client.Exec(`
			UPDATE users_aseguradores
			SET password_hash = $1,
				reset_token = NULL,
				reset_expires = NULL
			WHERE user_uuid = $2
			AND no_asesor = $3`, hashed_password, user_uuid, password_credentials.NumeroAsesor)
	}

	fmt.Printf(`Se cambiara la contraseña de: %s`, user_uuid)

	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	services.SendCustomMail(email, "Tu constraseña ha sido actualizada")

	services.HandleResponseSuccess(w)
}

func ApiRegisterUser(w http.ResponseWriter, r *http.Request) {
	/*
		 * http://127.0.0.1:3006/v1/auth/register
			* body:
			* {
			* 	"email": string,
			* 	"password":string
			* 	"insurance": string
			* }
	*/
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	fmt.Println("Request from ApiRegisterUser")
	fmt.Printf("----------------------------------------\n")

	var asegurador_data internal.UserAseguradorRegister

	err := json.NewDecoder(r.Body).Decode(&asegurador_data)
	if err != nil {
		fmt.Println("Error decoding JSON on ApiRegisterUser:", err)
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiRegisterUser", w)
		return
	}

	fmt.Printf(`Intento de registro para: %s`, asegurador_data.Email)

	hashed_password, err := services.HashPassword(asegurador_data.Password)
	if err != nil {
		fmt.Println("No se ha podido obtener el hash de la contraseña", err)
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido obtener el hash de la contraseña", w)
		return
	}

	verification_token, err := services.GenerateSecureToken()
	if err != nil {
		fmt.Println("Error al generar el token de verificación de cuenta", err)
		services.HandleResponseError(http.StatusBadRequest, "Error al generar el token de verificación de cuenta", w)
		return
	}

	_, err = db.Client.Exec(`
		INSERT INTO users_aseguradores (email, password_hash, fk_aseguradora, verification_token, verification_expires, no_asesor)
		VALUES ($1, $2, COALESCE($3, null), $4, $5, $6)`,
		asegurador_data.Email,
		hashed_password,
		asegurador_data.Insurance,
		verification_token,
		time.Now().Add(60*time.Minute),
		asegurador_data.NumeroAsesor)
	if err != nil {
		fmt.Println("No se ha podido realizar la inserción del usuario:", err)
		services.HandleResponseError(http.StatusBadRequest, fmt.Sprintf(`No se ha podido realizar la inserción del usuario: %s`, err.Error()), w)
		return
	}

	/**
	 * Aqui se debe de enviar el correo con el token de confirmación
	 */

	err = services.SendMail("oadrian38@gmail.com", verification_token, "Register")
	if err != nil {
		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, fmt.Sprintf("No se ha podido enviar el correo: %s", err.Error()), w)
		return
	}

	services.HandleResponseSuccess(w)
}

func ApiResetPasswordMail(w http.ResponseWriter, r *http.Request) {
	/*
		 * http://127.0.0.1:3006/v1/auth/resetpasswordmail
			* body:
			* {
			* 	"password":string
			* }
	*/
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	fmt.Println("Request from ApiResetPasswordMail")
	fmt.Printf("----------------------------------------\n")

	var reset_pass_credentials internal.ResetPasswordCredentials
	var email_obtained string

	err := json.NewDecoder(r.Body).Decode(&reset_pass_credentials)
	if err != nil {
		fmt.Println("Error decoding JSON on ApiResetPasswordMail:", err)
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiResetPasswordMail", w)
		return
	}

	err = db.Client.QueryRow(`
		SELECT email
		FROM users_aseguradores
		WHERE email = $1
		`, reset_pass_credentials.Email).
		Scan(&email_obtained)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err.Error())
			services.HandleResponseError(http.StatusUnauthorized, "No existe usuario", w)
			return
		}

		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error al consultar usuario", w)
		return
	}

	verification_token, err := services.GenerateSecureToken()
	if err != nil {
		fmt.Println("Error al generar token de restablecimiento", err)
		services.HandleResponseError(http.StatusInternalServerError, "Error al generar token de restablecimiento", w)
		return
	}

	_, err = db.Client.Exec(`
		UPDATE users_aseguradores
		SET reset_token = $1, reset_expires = $2
		WHERE email = $3`, verification_token, time.Now().Add(60*time.Minute), email_obtained)
	if err != nil {
		fmt.Println("No se ha podido insertar el token de restablecimiento", err)
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido insertar el token de restablecimiento", w)
		return
	}

	err = services.SendMail("oadrian38@gmail.com", verification_token, "ResetPassword")
	if err != nil {
		fmt.Println("No se ha podido enviar el correo de restablecimiento", err)
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido enviar el correo de restablecimiento", w)
		return
	}

	services.HandleResponseSuccess(w)
}

func ApiVerifyAccount(w http.ResponseWriter, r *http.Request) {
	/*
		 * http://127.0.0.1:3006/v1/auth/verifyaccount
			* body:
			* {
			* 	"token": string,
			* 	"password":string
			* 	"insurance": string
			* }
	*/
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	fmt.Println("Request from ApiVerifyAccount")
	fmt.Printf("----------------------------------------\n")

	var token_already_confirmed bool
	token := r.URL.Query().Get("token")

	fmt.Println("Iniciando validacion del token para confirmar la cuenta")

	rows, err := db.Client.Query(`SELECT is_verified FROM users_aseguradores WHERE verification_token = $1`, token)
	if err != nil {
		fmt.Println("No se han podido recuperar datos de verification_token", err)
		services.HandleResponseError(http.StatusBadRequest, "No se han podido recuperar datos de verification_token", w)
		return
	}

	for rows.Next() {
		if err := rows.Scan(&token_already_confirmed); err != nil {
			fmt.Println(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error al leer datos", w)
			return
		}
	}

	fmt.Println(token_already_confirmed)

	if token_already_confirmed == true {
		fmt.Println("La cuenta ya ha sido confirmada con anterioridad")
		services.HandleResponseError(http.StatusConflict, "La cuenta ya ha sido confirmada con anterioridad", w)
		return
	}

	fmt.Println(token)

	user_uuid, err := services.ValidateConfirmationToken(token)
	if err != nil {
		fmt.Println(err.Error())
		// services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		http.Redirect(w, r, "http://localhost:5173/auth/verifiedaccount?status=invalid", http.StatusSeeOther)
		return
	}

	_, err = db.Client.Exec(`
		UPDATE users_aseguradores
		SET is_verified = true,
			verification_expires = NULL,
			verification_token = NULL
		WHERE user_uuid = $1`, user_uuid)
	if err != nil {
		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, fmt.Sprintf("No se ha podido verificar la cuenta: %s", err.Error()), w)
		return
	}

	fmt.Println("Se ha verificado la cuenta")

	http.Redirect(w, r, "http://127.0.0.1:5173/auth/verifiedaccount?status=success", http.StatusSeeOther)
	return
}

/**
 *
 * Permite el origen que se haya recibido en el request
 * [] Globalizar la lista de los origenes permitidos
 *
 */
func AllowOrigins(w http.ResponseWriter, request *http.Request) {
	allowedOrigins := map[string]bool{
		"http://localhost:5173":                               true,
		"chrome-extension://jgahlmealgaocieaemladngafmbbfgdo": true,
	}

	origin := request.Header.Get("Origin")

	if allowedOrigins[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}
