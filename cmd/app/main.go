package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/configs"
	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/handlers"
)

func main() {
	dbConn, _ := db.CreateDbConn()
	db.Client = dbConn
	defer db.Client.Close()

	http.HandleFunc("/", configs.Serve)

	fmt.Printf("Servidor corriendo en http://%s:%v\n", env.APIHost, env.APIPort)
	fmt.Printf("----------------------------------------\n")

	handler := handlers.EnableCORS(http.DefaultServeMux)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(`:%s`, env.APIPort), handler))
}
