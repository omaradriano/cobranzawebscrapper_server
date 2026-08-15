package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/dto"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
	"gorm.io/gorm"
)

func ApiCheckPasswordExist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	var verifiedEmailRes dto.Verify_Password_Response
	email := chi.URLParam(r, "email")

	passwordHash, err := deps.AgenteRepo.FindPasswordByEmail(r.Context(), email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			services.HandleResponseError(http.StatusNotFound, "Usuario no encontrado", w)
			return
		}
		services.HandleResponseError(http.StatusInternalServerError, "Error al consultar password", w)
		return
	}

	verifiedEmailRes.HasPassword = passwordHash != nil && *passwordHash != ""
	if !verifiedEmailRes.HasPassword {
		verifiedEmailRes.PasswordToken, err = services.GenerateSecureToken()
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "No se ha podido generar token de contraseña", w)
			return
		}
		tokenExpires := time.Now().Add(30 * time.Minute)
		err = deps.AgenteRepo.UpdateResetToken(r.Context(), email, verifiedEmailRes.PasswordToken, tokenExpires)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error al guardar token de contraseña", w)
			return
		}
		services.HandleResponseSuccessWithData(verifiedEmailRes, w)
		return
	}

	services.HandleResponseSuccessWithData(verifiedEmailRes, w)
}

func ApiAuthenticateUserByGoogle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	var google_token dto.Google_Token

	err := json.NewDecoder(r.Body).Decode(&google_token)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	url := env.Envs.GoogleApiAuth
	payload := strings.NewReader(``)

	req, err := http.NewRequest("GET", url, payload)
	if err != nil {
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

	var googleUserResponse dto.Google_User_Response
	err = json.NewDecoder(resp.Body).Decode(&googleUserResponse)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "No existe respuesta de google_user_response", w)
		return
	}

	agente, aseguradora, err := deps.AgenteRepo.FindByEmailWithInsurance(r.Context(), googleUserResponse.Email)

	found := true
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			found = false
		} else {
			services.HandleResponseError(http.StatusInternalServerError, "No se ha podido hacer peticion de datos", w)
			return
		}
	}

	var ServerResponse dto.Server_Response_With_Token
	ServerResponse.Email = googleUserResponse.Email

	if !found {
		tempNoAgente := fmt.Sprintf("G-%04d", time.Now().UnixNano()%10000)
		aseguradoraID := int64(1)
		newAgente := &models.Agente{
			Email:         googleUserResponse.Email,
			IsVerified:    true,
			NoAgente:      tempNoAgente,
			Role:          "Agente",
			AseguradoraID: &aseguradoraID,
		}

		err = deps.AgenteRepo.Create(r.Context(), newAgente)
		if err != nil {
			services.HandleResponseError(http.StatusConflict, "No se ha podido crear un nuevo usuario por autenticacion google", w)
			return
		}

		ServerResponse.JWT_Token, err = services.GenerateJWT(
			newAgente.AgenteUUID.String(), ServerResponse.Email,
			newAgente.NoAgente, newAgente.Role, "Por asignar", "1",
		)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error al obtener jwt token para nuevo usuario", w)
			return
		}

		services.HandleResponseSuccessWithData(ServerResponse, w)
	} else {
		aseguradoraNombre := ""
		aseguradoraIDStr := ""
		if aseguradora != nil {
			aseguradoraNombre = aseguradora.Nombre
			aseguradoraIDStr = strconv.Itoa(aseguradora.AseguradoraID)
		}

		ServerResponse.JWT_Token, err = services.GenerateJWT(
			agente.AgenteUUID.String(), agente.Email,
			agente.NoAgente, agente.Role, aseguradoraNombre, aseguradoraIDStr,
		)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error al obtener jwt token", w)
			return
		}

		w.Header().Set("Authorization", fmt.Sprintf(`Bearer %s`, ServerResponse.JWT_Token))
		services.HandleResponseSuccessWithData(ServerResponse, w)
	}
}

func ApiAuthenticateUserByCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	var loginCredentials dto.LoginUserCredentials
	var ServerResponse dto.Server_Response_With_Token

	err := json.NewDecoder(r.Body).Decode(&loginCredentials)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiAuthenticateUserByCredentials", w)
		return
	}

	agente, aseguradora, err := deps.AgenteRepo.FindByEmailWithInsurance(r.Context(), loginCredentials.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			services.HandleResponseError(http.StatusUnauthorized, "Credenciales incorrectas", w)
			return
		}
		services.HandleResponseError(http.StatusInternalServerError, "Error al consultar usuario", w)
		return
	}

	if !agente.IsVerified {
		services.HandleResponseError(http.StatusUnauthorized, "La cuenta no esta verificada", w)
		return
	}

	if agente.PasswordHash == nil || !services.CheckPassword(*agente.PasswordHash, loginCredentials.Password) {
		services.HandleResponseError(http.StatusUnauthorized, "Credenciales incorrectas", w)
		return
	}

	aseguradoraNombre := ""
	aseguradoraIDStr := ""
	if aseguradora != nil {
		aseguradoraNombre = aseguradora.Nombre
		aseguradoraIDStr = strconv.Itoa(aseguradora.AseguradoraID)
	}

	ServerResponse.JWT_Token, err = services.GenerateJWT(
		agente.AgenteUUID.String(), agente.Email,
		agente.NoAgente, agente.Role, aseguradoraNombre, aseguradoraIDStr,
	)

	services.HandleResponseSuccessWithData(ServerResponse, w)
}

func ApiCheckSession(w http.ResponseWriter, r *http.Request) {
	var sessionClaims dto.JWTClaims

	sessionClaims.AgenteUUID, _ = r.Context().Value(middlewares.UserIDKey).(string)
	sessionClaims.Email, _ = r.Context().Value(middlewares.UserEmailKey).(string)
	sessionClaims.NoAgente, _ = r.Context().Value(middlewares.UserNoAgente).(string)
	sessionClaims.Role, _ = r.Context().Value(middlewares.UserRole).(string)
	sessionClaims.InsuranceName, _ = r.Context().Value(middlewares.UserInsurance).(string)
	sessionClaims.InsuranceID, _ = r.Context().Value(middlewares.UserInsuranceID).(string)

	services.HandleResponseSuccessWithData(sessionClaims, w)
}

func ApiSetCredentials(w http.ResponseWriter, r *http.Request) {
	var passwordCredentials dto.SetPasswordCredentials

	passwordCredentials.ResetToken = r.URL.Query().Get("token")

	err := json.NewDecoder(r.Body).Decode(&passwordCredentials)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiSetPassword", w)
		return
	}

	userUUID, email, err := services.ValidateResetToken(passwordCredentials.ResetToken)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	agente, err := deps.AgenteRepo.FindByUUID(r.Context(), userUUID)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	hashedPassword, err := services.HashPassword(passwordCredentials.Password)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error al generar hash de contraseña", w)
		return
	}

	var noAgente *string
	var aseguradoraID *string
	if agente.NoAgente == "000000" || strings.HasPrefix(agente.NoAgente, "G-") {
		noAgente = &passwordCredentials.NumeroAsesor
		aseguradoraID = &passwordCredentials.Aseguradora
	}

	err = deps.AgenteRepo.UpdatePassword(r.Context(), userUUID, hashedPassword, noAgente, aseguradoraID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "23505") || strings.Contains(errMsg, "unique") {
			services.Log.ErrorMessage("Error 23505: El número de asesor ya está registrado por otro usuario.")
			services.HandleResponseError(http.StatusConflict, "El número de asesor ya se encuentra en uso por otra cuenta.", w)
			return
		}
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
		return
	}

	services.SendCustomMail(email, "Tu constraseña ha sido actualizada")
	services.HandleResponseSuccess(w)
}

func ApiRegisterUser(w http.ResponseWriter, r *http.Request) {
	var aseguradorData dto.UserAseguradorRegister

	err := json.NewDecoder(r.Body).Decode(&aseguradorData)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiRegisterUser", w)
		return
	}

	hashedPassword, err := services.HashPassword(aseguradorData.Password)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido obtener el hash de la contraseña", w)
		return
	}

	verificationToken, err := services.GenerateSecureToken()
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "Error al generar el token de verificación de cuenta", w)
		return
	}

	verificationExpires := time.Now().Add(30 * 24 * time.Hour)

	var aseguradoraID *int64
	if aseguradorData.Insurance != nil {
		id, parseErr := strconv.ParseInt(*aseguradorData.Insurance, 10, 64)
		if parseErr == nil {
			aseguradoraID = &id
		}
	}

	newAgente := &models.Agente{
		Email:               aseguradorData.Email,
		PasswordHash:        &hashedPassword,
		AseguradoraID:       aseguradoraID,
		VerificationToken:   &verificationToken,
		VerificationExpires: &verificationExpires,
		NoAgente:            aseguradorData.NumeroAsesor,
	}

	err = deps.AgenteRepo.Create(r.Context(), newAgente)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, fmt.Sprintf("No se ha podido realizar la inserción del usuario: %s", err.Error()), w)
		return
	}

	err = services.SendMail(aseguradorData.Email, verificationToken, "Register")
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, fmt.Sprintf("No se ha podido enviar el correo: %s", err.Error()), w)
		return
	}

	services.HandleResponseSuccess(w)
}

func ApiResetPasswordMail(w http.ResponseWriter, r *http.Request) {
	var resetPassCredentials dto.ResetPasswordCredentials

	err := json.NewDecoder(r.Body).Decode(&resetPassCredentials)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "No se ha podido recuperar formato json de ApiResetPasswordMail", w)
		return
	}

	_, err = deps.AgenteRepo.FindEmailByEmail(r.Context(), resetPassCredentials.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			services.HandleResponseError(http.StatusUnauthorized, "No existe usuario", w)
			return
		}
		services.HandleResponseError(http.StatusInternalServerError, "Error al consultar usuario", w)
		return
	}

	verificationToken, err := services.GenerateSecureToken()
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error al generar token de restablecimiento", w)
		return
	}

	err = deps.AgenteRepo.UpdateResetToken(r.Context(), resetPassCredentials.Email, verificationToken, time.Now().Add(60*time.Minute))
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido insertar el token de restablecimiento", w)
		return
	}

	err = services.SendMail(resetPassCredentials.Email, verificationToken, "ResetPassword")
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido enviar el correo de restablecimiento", w)
		return
	}

	services.HandleResponseSuccess(w)
}

func ApiVerifyAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	token := r.URL.Query().Get("token")
	if token == "" {
		redirectUrl := fmt.Sprintf(`http://%s/auth/verifiedaccount?status=invalid`, env.Envs.WebAppURL)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	userUUID, err := services.ValidateConfirmationToken(token)
	if err != nil {
		redirectUrl := fmt.Sprintf(`%s/auth/verifiedaccount?status=invalid`, env.Envs.WebAppURL)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	isVerified, err := deps.AgenteRepo.FindIsVerified(r.Context(), token, userUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			redirectUrl := fmt.Sprintf(`%s/auth/verifiedaccount?status=invalid`, env.Envs.WebAppURL)
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
			return
		}
		services.HandleResponseError(http.StatusInternalServerError, "Error interno del servidor", w)
		return
	}

	if isVerified {
		redirectUrl := fmt.Sprintf(`%s/auth/verifiedaccount?status=already_confirmed`, env.Envs.WebAppURL)
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}

	err = deps.AgenteRepo.UpdateVerification(r.Context(), userUUID)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se ha podido verificar la cuenta debido a un error interno", w)
		return
	}

	redirectUrl := fmt.Sprintf(`http://%s/auth/verifiedaccount?status=success`, env.Envs.WebAppURL)
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}
