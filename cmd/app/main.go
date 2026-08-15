package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/configs"
	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/handlers"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/repository"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
	"github.com/stripe/stripe-go/v74"
)

func main() {
	env.LoadConfig()
	middlewares.JwtSecret = env.Envs.JWTSecret
	stripe.Key = env.Envs.StripeSecret

	gormConn, err := db.CreateGormConn()
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		return
	}
	db.GormDB = gormConn

	handlers.InitDeps(gormConn)
	db.SetTokenValidator(repository.NewAgenteRepository(gormConn))

	router := configs.NewRouter()

	fmt.Printf("Servidor corriendo en http://%s:%v\n", env.Envs.ServerHost, env.Envs.ServerPort)
	fmt.Printf("----------------------------------------\n")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(`:%s`, env.Envs.ServerPort), router))
}
