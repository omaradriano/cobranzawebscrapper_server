package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"golang.org/x/crypto/bcrypt"
)

func HandleResponseError(Code int, Message string, w http.ResponseWriter) {
	customErr := &internal.HttpError{
		Code:    Code,
		Message: Message,
		Success: false,
	}

	jsonResponse, err := json.Marshal(customErr)
	if err != nil {
		NewLogger().ErrorMessage(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(Code)
	w.Write(jsonResponse)
}

func HandleResponseSuccessWithData(Payload interface{}, w http.ResponseWriter) {
	customSuccess := &internal.HttpSuccess{
		Code:    http.StatusOK,
		Payload: Payload,
		Success: true,
	}

	jsonResponse, err := json.Marshal(customSuccess)
	if err != nil {
		NewLogger().ErrorMessage(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(jsonResponse)
}

func HandleResponseSuccess(w http.ResponseWriter) {
	customSuccess := &internal.HttpSuccess{
		Code:    http.StatusAccepted,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusAccepted)

	json.NewEncoder(w).Encode(customSuccess)
}

func GenerateJWT(user_uuid, email, no_agente, role, aseguradora, aseguradora_id string) (string, error) {
	claims := jwt.MapClaims{
		"uuid":           user_uuid,
		"exp":            time.Now().Add(time.Hour * 24).Unix(),
		"email":          email,
		"role":           role,
		"no_agente":      no_agente,
		"insurance_name": aseguradora,
		"insurance_id":   aseguradora_id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(middlewares.JwtSecret))
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(middlewares.JwtSecret), nil
	})
}

func GenerateSecureToken() (string, error) {
	b := make([]byte, 32) // 256 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ValidateResetToken(token string) (string, string, error) {
	var user_uuid string
	var email string
	var no_agente string
	var expires time.Time

	err := db.Client.QueryRow(`
		SELECT agente_uuid, email, reset_expires, no_agente
		FROM agentes
		WHERE reset_token = $1
	`, token).Scan(&user_uuid, &email, &expires, &no_agente)
	if err != nil {
		fmt.Println(err.Error())
		return "", "", fmt.Errorf("token inválido")
	}

	if time.Now().After(expires) {
		return "", "", fmt.Errorf("token expirado")
	}

	return user_uuid, email, nil
}

func ValidateConfirmationToken(token string) (string, error) {
	var user_uuid string
	var expires time.Time

	err := db.Client.QueryRow(`
		SELECT agente_uuid, verification_expires
		FROM agentes
		WHERE verification_token = $1
	`, token).Scan(&user_uuid, &expires)
	if err != nil {
		return "", fmt.Errorf("token inválido")
	}

	if time.Now().After(expires) {
		return "", fmt.Errorf("token expirado")
	}

	return user_uuid, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:5173" ||
			origin == "https://www.goagent.com.mx" ||
			origin == "https://goagent.com.mx" ||
			origin == "chrome-extension://jgahlmealgaocieaemladngafmbbfgdo" {

			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
