package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/configs"
	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

func main() {
	env.LoadConfig()
	dbConn, err := db.CreateDbConn()
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		return
	}

	db.Client = dbConn
	defer db.Client.Close()

	http.HandleFunc("/", configs.Serve)

	fmt.Printf("Servidor corriendo en http://%s:%v\n", env.Envs.ServerHost, env.Envs.ServerPort)
	fmt.Printf("----------------------------------------\n")

	handler := services.EnableCORS(http.DefaultServeMux)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(`:%s`, env.Envs.ServerPort), handler))
}
