package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

func GetField(r *http.Request, index int) string {
	fields := r.Context().Value(internal.CtxKey{}).([]string)
	return fields[index]
}

func ApiSetPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PATCH")

	fmt.Println("Request from ApiPatchPayment")
	fmt.Printf("----------------------------------------\n")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var item internal.CobranzaItemPayment

	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		fmt.Println("ola")
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	err = db.Client.QueryRow(`SELECT agente_id FROM agentes WHERE agente_uuid = $1`, userUUID).Scan(&item.Agente)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}
	err = db.Client.QueryRow(`SELECT poliza_id FROM polizas WHERE poliza_uuid = $1`, item.Poliza).Scan(&item.Poliza)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}

	vQuery := `	INSERT INTO polizas_payments_log (poliza_id, agente_id, paid_period)
				VALUES ($1, $2, $3)`
	_, err = db.Client.Exec(vQuery, item.Poliza, item.Agente, item.PaidPeriod)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}
	services.HandleResponseSuccess(w)
}

func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:5173" ||
			origin == "https://www.goagent.com.mx" ||
			origin == "https://goagent.com.mx" {

			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
