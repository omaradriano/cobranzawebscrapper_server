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
	"github.com/omaradriano/cobranzawebscrapper_server/internal/dto"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"golang.org/x/crypto/bcrypt"
)

func HandleResponseError(Code int, Message string, w http.ResponseWriter) {
	customErr := &dto.HttpError{
		Code:    Code,
		Message: Message,
		Success: false,
	}

	jsonResponse, err := json.Marshal(customErr)
	if err != nil {
		Log.ErrorMessage(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(Code)
	w.Write(jsonResponse)
}

func HandleResponseSuccessWithData(Payload interface{}, w http.ResponseWriter) {
	customSuccess := &dto.HttpSuccess{
		Code:    http.StatusOK,
		Payload: Payload,
		Success: true,
	}

	jsonResponse, err := json.Marshal(customSuccess)
	if err != nil {
		Log.ErrorMessage(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func HandleResponseSuccess(w http.ResponseWriter) {
	customSuccess := &dto.HttpSuccess{
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
	uuid, email, _, expires, err := db.AgenteRepo.ValidateResetToken(token)
	if err != nil {
		return "", "", fmt.Errorf("token inválido")
	}

	if time.Now().After(expires) {
		return "", "", fmt.Errorf("token expirado")
	}

	return uuid, email, nil
}

func ValidateConfirmationToken(token string) (string, error) {
	uuid, expires, err := db.AgenteRepo.ValidateConfirmationToken(token)
	if err != nil {
		return "", fmt.Errorf("token inválido")
	}

	if time.Now().After(expires) {
		return "", fmt.Errorf("token expirado")
	}

	return uuid, nil
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
	return CORSMiddleware(next)
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:5173" ||
			origin == "https://www.goagent.com.mx" ||
			origin == "https://goagent.com.mx" ||
			origin == "chrome-extension://bnhggcmlbinhmheijhmlfjoefgldkpdp" ||
			origin == "chrome-extension://jgahlmealgaocieaemladngafmbbfgdo" ||
			origin == "chrome-extension://acihafkligkgjbhmbgaidkackhojbokh" {

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
