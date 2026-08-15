package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/omaradriano/cobranzawebscrapper_server/env"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/dto"
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

	priceID := env.Envs.StripePriceID

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

	endpointSecret := env.Envs.StripeWebhookSign
	signatureHeader := r.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(payload, signatureHeader, endpointSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		services.Log.ErrorMessage("Firma del Webhook inválida: " + err.Error())
		services.HandleResponseError(http.StatusBadRequest, "Firma inválida", w)
		return
	}

	ctx := r.Context()

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

		if err = deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
			"is_subscribed":          true,
			"stripe_subscription_id": subscriptionID,
		}, "agente_uuid = ?", agenteUUID); err != nil {
			services.Log.ErrorMessage("Error activando suscripción del agente " + agenteUUID + ": " + err.Error())
			services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
			return
		}

		if subscriptionID != "" {
			sub, subErr := stripesubscription.Get(subscriptionID, nil)
			if subErr != nil {
				services.Log.ErrorMessage("Error obteniendo suscripción " + subscriptionID + ": " + subErr.Error())
			} else {
				if dbErr := deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
					"cancel_at_period_end": sub.CancelAtPeriodEnd,
					"current_period_end":   sub.CurrentPeriodEnd,
				}, "agente_uuid = ?", agenteUUID); dbErr != nil {
					services.Log.ErrorMessage("Error guardando current_period_end: " + dbErr.Error())
				}
			}
		}

	case "customer.subscription.updated":
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

		if sub.CurrentPeriodEnd == 0 {
			break
		}

		if err = deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
			"cancel_at_period_end": sub.CancelAtPeriodEnd,
			"current_period_end":   sub.CurrentPeriodEnd,
		}, "stripe_subscription_id = ?", sub.ID); err != nil {
			services.Log.ErrorMessage("Error actualizando estado de suscripción " + sub.ID + ": " + err.Error())
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

		if err = deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
			"is_subscribed": true,
		}, "stripe_subscription_id = ?", invoice.Subscription.ID); err != nil {
			services.Log.ErrorMessage("Error renovando suscripción " + invoice.Subscription.ID + ": " + err.Error())
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

		if err = deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
			"is_subscribed": false,
		}, "stripe_subscription_id = ?", invoice.Subscription.ID); err != nil {
			services.Log.ErrorMessage("Error desactivando suscripción por fallo de cobro " + invoice.Subscription.ID + ": " + err.Error())
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

		if err = deps.AgenteRepo.UpdateSubscription(ctx, map[string]any{
			"is_subscribed":          false,
			"cancel_at_period_end":   false,
			"current_period_end":     0,
			"stripe_subscription_id": nil,
		}, "stripe_subscription_id = ?", sub.ID); err != nil {
			services.Log.ErrorMessage("Error desactivando suscripción eliminada " + sub.ID + ": " + err.Error())
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

	agente, err := deps.AgenteRepo.GetSubscriptionStatus(r.Context(), agente_uuid)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error consultando suscripción", w)
		return
	}

	payload := dto.SubscriptionStatusPayload{
		IsSubscribed:      agente.IsSubscribed,
		CancelAtPeriodEnd: agente.CancelAtPeriodEnd,
		CurrentPeriodEnd:  agente.CurrentPeriodEnd,
	}

	services.HandleResponseSuccessWithData(payload, w)
}

func ApiCancelSubscription(w http.ResponseWriter, r *http.Request) {
	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)
	if agente_uuid == "" {
		services.HandleResponseError(http.StatusUnauthorized, "Usuario no autenticado", w)
		return
	}

	subscriptionID, err := deps.AgenteRepo.GetSubscriptionID(r.Context(), agente_uuid)
	if err != nil || subscriptionID == "" {
		services.HandleResponseError(http.StatusBadRequest, "No se encontró una suscripción activa", w)
		return
	}

	updatedSub, err := stripesubscription.Update(subscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	if err != nil {
		services.Log.ErrorMessage("Error programando cancelación en Stripe " + subscriptionID + ": " + err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error cancelando suscripción en Stripe", w)
		return
	}

	if err = deps.AgenteRepo.UpdateSubscription(r.Context(), map[string]any{
		"cancel_at_period_end": updatedSub.CancelAtPeriodEnd,
		"current_period_end":   updatedSub.CurrentPeriodEnd,
	}, "agente_uuid = ?", agente_uuid); err != nil {
		services.Log.ErrorMessage("Error actualizando cancelación en DB " + subscriptionID + ": " + err.Error())
		services.HandleResponseError(http.StatusInternalServerError, "Error interno actualizando DB", w)
		return
	}

	services.HandleResponseSuccessWithData(map[string]any{
		"cancel_at_period_end": updatedSub.CancelAtPeriodEnd,
		"current_period_end":   updatedSub.CurrentPeriodEnd,
	}, w)
}
