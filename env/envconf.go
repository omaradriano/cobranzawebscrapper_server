package env

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/joho/godotenv"
)

type Config struct {
	WebAppURL  string
	WebAppPort string

	Db_Database string
	Db_User     string
	Db_Password string
	Db_Port     string
	Db_Server   string

	Mode string

	JWTSecret string

	ResendToken string

	ServerHost string
	ServerPort string

	GoogleApiAuth string
}

var Envs *Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: No se encontró el archivo .env, usando variables del sistema operativo.")
	}

	Envs = &Config{
		WebAppURL:   getEnv("WEBAPP_URL", ""),
		WebAppPort:  getEnv("WEBAPP_PORT", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		Mode:        getEnv("MODE", ""),
		Db_Database: getEnv("DB_DATABASE", ""),
		Db_User:     getEnv("DB_USER", ""),
		Db_Password: getEnv("DB_PASSWORD", ""),
		Db_Port:     getEnv("DB_PORT", ""),
		Db_Server:   getEnv("DB_SERVER", ""),

		ResendToken: getEnv("TOKEN_RESEND", ""),

		ServerHost: getEnv("SERVER_HOST", ""),
		ServerPort: getEnv("SERVER_PORT", ""),

		GoogleApiAuth: getEnv("GOOGLE_API_AUTH_URL", ""),
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
			log.Fatalf("Falta una variable de entorno. Verificar: %s", field)
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
