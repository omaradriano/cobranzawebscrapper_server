package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
)

var Client *sql.DB

// Esto es lo que contiene la conexion general a la base de datos
type Env struct {
	DbClient *sql.DB
}

func CreateDbConn() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		env.PostgresServer, env.PostgresDatabasePort, env.PostgresUser, env.PostgresPassword, env.PostgresDatabase)

	Client, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error al abrir la definición de la conexión:", err)
		return nil, err
	}

	err = Client.Ping()
	if err != nil {
		fmt.Println("No se pudo conectar a Postgres:", err)
		return nil, err
	}

	return Client, nil
}
