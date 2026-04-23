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

func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"email":   "ejemploEmail",
		"role":    "admin",
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

func ValidateResetToken(token string) (string, error) {
	var user_uuid string
	var expires time.Time

	err := db.Client.QueryRow(`
		SELECT user_uuid, reset_expires
		FROM users_aseguradores
		WHERE reset_token = $1
	`, token).Scan(&user_uuid, &expires)
	if err != nil {
		return "", fmt.Errorf("token inválido")
	}

	if time.Now().After(expires) {
		return "", fmt.Errorf("token expirado")
	}

	return user_uuid, nil
}

func ValidateConfirmationToken(token string) (string, error) {
	var user_uuid string
	var expires time.Time

	err := db.Client.QueryRow(`
		SELECT user_uuid, verification_expires
		FROM users_aseguradores
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
