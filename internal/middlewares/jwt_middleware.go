package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

var JwtSecret = `perryelortitorico1389`

const (
	UserIDKey    contextKey = "agente_uuid"
	UserEmailKey contextKey = "email"
	UserNoAgente contextKey = "no_agente"
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, "Token requerido", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			http.Error(w, "Formato de token inválido", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
			}

			return []byte(JwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Claims inválidos", http.StatusUnauthorized)
			return
		}

		// Verificación de expiración
		if exp, ok := claims["exp"].(float64); ok {
			if int64(exp) < time.Now().Unix() {
				http.Error(w, "Token expirado", http.StatusUnauthorized)
				return
			}
		}

		// Extraer claims
		uuid, _ := claims["agente_uuid"].(string)
		email, _ := claims["email"].(string)
		agente, _ := claims["no_agente"].(string)

		// Guardar ambos en el contexto
		ctx := context.WithValue(r.Context(), UserIDKey, uuid)
		ctx = context.WithValue(ctx, UserEmailKey, email)
		ctx = context.WithValue(ctx, UserNoAgente, agente)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
