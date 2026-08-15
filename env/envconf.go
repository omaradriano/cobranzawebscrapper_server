package env

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/joho/godotenv"
)

type Config struct {
	WebAppURL string
	DB_URL    string

	Mode string

	JWTSecret string

	ResendToken string

	ServerHost string
	ServerPort string

	GoogleApiAuth string

	MailDestinationWeb    string
	MailDestinationServer string

	StripeSecret      string
	StripeWebhookSign string
	StripeRedirectUrl string
	StripePriceID     string
}

var Envs *Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: No se encontró el archivo .env, usando variables del sistema operativo.")
	}

	Envs = &Config{
		WebAppURL: getEnv("WEBAPP_URL", ""),
		JWTSecret: getEnv("JWT_SECRET", ""),
		Mode:      getEnv("MODE", ""),
		DB_URL:    getEnv("DB_URL", ""),

		ResendToken: getEnv("TOKEN_RESEND", ""),

		ServerHost: getEnv("SERVER_HOST", ""),
		ServerPort: getEnv("SERVER_PORT", ""),

		GoogleApiAuth: getEnv("GOOGLE_API_AUTH_URL", ""),

		MailDestinationWeb:    getEnv("MAIL_DESTINATION_WEB", ""),
		MailDestinationServer: getEnv("MAIL_DESTINATION_SERVER", ""),

		StripeSecret: getEnv("STRIPE_SECRET", ""),

		StripeWebhookSign: getEnv("STRIPE_WEBHOOK_SECRET", ""),

		StripeRedirectUrl: getEnv("STRIPE_REDIRECT_URL", ""),

		StripePriceID: getEnv("STRIPE_PRICE_ID", ""),
	}

	valueof := reflect.ValueOf(Envs).Elem()
	typeof := valueof.Type()

	if typeof.Kind() != reflect.Struct {
		fmt.Println("Not a struct. Check your input!")
		return
	}

	for i := 0; i < typeof.NumField(); i++ {
		field := typeof.Field(i)
		value := valueof.Field(i)
		if fmt.Sprintf("%s", value) == "" {
			log.Fatalf("Falta una variable de entorno. Verificar: %s", field.Name)
		}
	}
}

// Función auxiliar para leer variables o retornar un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
