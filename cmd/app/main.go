package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/configs"
)

func main() {
	port := ":3006"

	http.HandleFunc("/", configs.Serve) // <---- entry point

	fmt.Printf("Servidor corriendo en http://localhost%v\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
