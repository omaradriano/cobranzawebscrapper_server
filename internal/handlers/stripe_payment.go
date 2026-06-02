package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/webhook"
)

type CheckoutRequest struct {
	Plan string `json:"plan"`
}

func CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)

	if agente_uuid == "" {
		services.HandleResponseError(http.StatusUnauthorized, "Usuario no autenticado", w)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Datos invalidos", w)
		return
	}

	priceID := "price_1TcWxRLU4Gfljtd5YubSNHYO"

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String("http://localhost:5173/success_payment"),
		CancelURL:  stripe.String("http://localhost:5173/pricing"),
	}

	params.AddMetadata("agente_uuid", agente_uuid)

	s, err := session.New(params)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "No se pudo crear la sesión de pago", w)
		return
	}

	services.HandleResponseSuccessWithData(map[string]string{
		"url": s.URL,
	}, w)
}

func StripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		services.HandleResponseError(http.StatusBadRequest, "Error leyendo payload", w)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	signatureHeader := r.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(payload, signatureHeader, endpointSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		services.NewLogger().ErrorMessage("Firma del Webhook inválida: " + err.Error())
		services.HandleResponseError(http.StatusBadRequest, "Firma inválida", w)
		return
	}

	if event.Type == "checkout.session.completed" {
		sessionBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando datos de Stripe", w)
			return
		}

		var stripeSession stripe.CheckoutSession
		err = json.Unmarshal(sessionBytes, &stripeSession)
		if err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando estructura de la sesión", w)
			return
		}

		agenteUUID := stripeSession.Metadata["agente_uuid"]

		fmt.Println(agenteUUID)
		fmt.Println(`Ha completado un pago`)

		if agenteUUID != "" {
			_, err = db.Client.Exec(
				`UPDATE agentes SET is_subscribed = true WHERE agente_uuid = $1`,
				agenteUUID,
			)
			if err != nil {
				services.NewLogger().ErrorMessage("Error actualizando suscripción del agente " + agenteUUID + ": " + err.Error())
				services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func ApiGetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)
	if agente_uuid == "" {
		services.HandleResponseError(http.StatusUnauthorized, "Usuario no autenticado", w)
		return
	}

	var isSubscribed bool
	err := db.Client.QueryRow(
		`SELECT is_subscribed FROM agentes WHERE agente_uuid = $1`, agente_uuid,
	).Scan(&isSubscribed)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error consultando suscripción", w)
		return
	}

	services.HandleResponseSuccessWithData(map[string]bool{
		"is_subscribed": isSubscribed,
	}, w)
}
