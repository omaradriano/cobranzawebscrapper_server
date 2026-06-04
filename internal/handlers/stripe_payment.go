package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	stripesubscription "github.com/stripe/stripe-go/v74/subscription"
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

	priceID := "price_1TeQ9sLU4Gfljtd5qQCKmVwx"

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
		SuccessURL: stripe.String(fmt.Sprintf(`http://%s/success_payment`, env.Envs.StripeRedirectUrl)),
		CancelURL:  stripe.String(fmt.Sprintf(`http://%s/pricing`, env.Envs.StripeRedirectUrl)),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"agente_uuid": agente_uuid,
			},
		},
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

	switch event.Type {

	case "checkout.session.completed":
		sessionBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando datos de Stripe", w)
			return
		}

		var stripeSession stripe.CheckoutSession
		if err = json.Unmarshal(sessionBytes, &stripeSession); err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando estructura de la sesión", w)
			return
		}

		agenteUUID := stripeSession.Metadata["agente_uuid"]
		if agenteUUID == "" {
			break
		}

		subscriptionID := ""
		if stripeSession.Subscription != nil {
			subscriptionID = stripeSession.Subscription.ID
		}

		// Guardar is_subscribed y stripe_subscription_id
		if _, err = db.Client.Exec(
			`UPDATE agentes SET is_subscribed = true, stripe_subscription_id = $1 WHERE agente_uuid = $2`,
			subscriptionID, agenteUUID,
		); err != nil {
			services.NewLogger().ErrorMessage("Error activando suscripción del agente " + agenteUUID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
		}

		// Obtener current_period_end desde Stripe y guardarlo inmediatamente
		if subscriptionID != "" {
			stripe.Key = os.Getenv("STRIPE_SECRET")
			sub, subErr := stripesubscription.Get(subscriptionID, nil)
			if subErr != nil {
				services.NewLogger().ErrorMessage("Error obteniendo suscripción " + subscriptionID + ": " + subErr.Error())
			} else {
				fmt.Printf("[DEBUG] stripe.Get OK — subscriptionID: %s | CurrentPeriodEnd: %d | CancelAtPeriodEnd: %v | agente: %s\n",
					subscriptionID, sub.CurrentPeriodEnd, sub.CancelAtPeriodEnd, agenteUUID)
				result, dbErr := db.Client.Exec(
					`UPDATE agentes SET cancel_at_period_end = $1, current_period_end = $2 WHERE agente_uuid = $3`,
					sub.CancelAtPeriodEnd, sub.CurrentPeriodEnd, agenteUUID,
				)
				if dbErr != nil {
					services.NewLogger().ErrorMessage("Error guardando current_period_end: " + dbErr.Error())
				} else {
					rows, _ := result.RowsAffected()
					fmt.Printf("[DEBUG] current_period_end update — rows afectadas: %d\n", rows)
				}
			}
		}

	case "customer.subscription.updated":
		// Maneja renovaciones y cancelaciones (no la creación inicial, que llega antes de checkout.session.completed)
		subBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando suscripción", w)
			return
		}

		var sub stripe.Subscription
		if err = json.Unmarshal(subBytes, &sub); err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando suscripción", w)
			return
		}

		fmt.Printf("[DEBUG] subscription.updated — ID: %s | CurrentPeriodEnd: %d | CancelAtPeriodEnd: %v\n",
			sub.ID, sub.CurrentPeriodEnd, sub.CancelAtPeriodEnd)

		if sub.CurrentPeriodEnd == 0 {
			fmt.Printf("[DEBUG] subscription.updated — CurrentPeriodEnd es 0, omitiendo update para no sobreescribir\n")
			break
		}

		if _, err = db.Client.Exec(
			`UPDATE agentes SET cancel_at_period_end = $1, current_period_end = $2 WHERE stripe_subscription_id = $3`,
			sub.CancelAtPeriodEnd, sub.CurrentPeriodEnd, sub.ID,
		); err != nil {
			services.NewLogger().ErrorMessage("Error actualizando estado de suscripción " + sub.ID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
		}

	case "invoice.payment_succeeded":
		invoiceBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando invoice", w)
			return
		}

		var invoice stripe.Invoice
		if err = json.Unmarshal(invoiceBytes, &invoice); err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando invoice", w)
			return
		}

		if invoice.Subscription == nil {
			break
		}

		if _, err = db.Client.Exec(
			`UPDATE agentes SET is_subscribed = true WHERE stripe_subscription_id = $1`,
			invoice.Subscription.ID,
		); err != nil {
			services.NewLogger().ErrorMessage("Error renovando suscripción " + invoice.Subscription.ID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
		}

	case "invoice.payment_failed":
		invoiceBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando invoice", w)
			return
		}

		var invoice stripe.Invoice
		if err = json.Unmarshal(invoiceBytes, &invoice); err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando invoice", w)
			return
		}

		if invoice.Subscription == nil {
			break
		}

		if _, err = db.Client.Exec(
			`UPDATE agentes SET is_subscribed = false WHERE stripe_subscription_id = $1`,
			invoice.Subscription.ID,
		); err != nil {
			services.NewLogger().ErrorMessage("Error desactivando suscripción por fallo de cobro " + invoice.Subscription.ID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
		}

	case "customer.subscription.deleted":
		subBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			services.HandleResponseError(http.StatusInternalServerError, "Error serializando suscripción", w)
			return
		}

		var sub stripe.Subscription
		if err = json.Unmarshal(subBytes, &sub); err != nil {
			services.HandleResponseError(http.StatusBadRequest, "Error mapeando suscripción", w)
			return
		}

		if _, err = db.Client.Exec(
			`UPDATE agentes SET is_subscribed = false, cancel_at_period_end = false, current_period_end = 0, stripe_subscription_id = NULL WHERE stripe_subscription_id = $1`,
			sub.ID,
		); err != nil {
			services.NewLogger().ErrorMessage("Error desactivando suscripción eliminada " + sub.ID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
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

	var payload internal.SubscriptionStatusPayload
	err := db.Client.QueryRow(
		`SELECT is_subscribed, cancel_at_period_end, current_period_end FROM agentes WHERE agente_uuid = $1`,
		agente_uuid,
	).Scan(&payload.IsSubscribed, &payload.CancelAtPeriodEnd, &payload.CurrentPeriodEnd)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error consultando suscripción", w)
		return
	}

	services.HandleResponseSuccessWithData(payload, w)
}

func ApiCancelSubscription(w http.ResponseWriter, r *http.Request) {
	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)
	if agente_uuid == "" {
		services.HandleResponseError(http.StatusUnauthorized, "Usuario no autenticado", w)
		return
	}

	var subscriptionID string
	err := db.Client.QueryRow(
		`SELECT COALESCE(stripe_subscription_id, '') FROM agentes WHERE agente_uuid = $1`, agente_uuid,
	).Scan(&subscriptionID)
	if err != nil || subscriptionID == "" {
		services.HandleResponseError(http.StatusBadRequest, "No se encontró una suscripción activa", w)
		return
	}

	updatedSub, err := stripesubscription.Update(subscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	if err != nil {
		services.NewLogger().ErrorMessage("Error programando cancelación en Stripe " + subscriptionID + ": " + err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error cancelando suscripción en Stripe", w)
		return
	}
	if _, err = db.Client.Exec(
		`UPDATE agentes SET cancel_at_period_end = $1, current_period_end = $2 WHERE agente_uuid = $3`,
		updatedSub.CancelAtPeriodEnd, updatedSub.CurrentPeriodEnd, agente_uuid,
	); err != nil {
		services.NewLogger().ErrorMessage("Error actualizando cancelación en DB " + subscriptionID + ": " + err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
		return
	}

	services.HandleResponseSuccessWithData(map[string]interface{}{
		"cancel_at_period_end": updatedSub.CancelAtPeriodEnd,
		"current_period_end":   updatedSub.CurrentPeriodEnd,
	}, w)
}
