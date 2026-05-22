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
		env.Envs.Db_Server, env.Envs.Db_Port, env.Envs.Db_User,
		env.Envs.Db_Password, env.Envs.Db_Database)

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
