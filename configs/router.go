package configs

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/handlers"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middlewares.RateLimitMiddleware)
	r.Use(services.CORSMiddleware)

	r.Route("/v1", func(r chi.Router) {
		// Auth routes (no JWT required)
		r.Post("/auth/authenticate/google", handlers.ApiAuthenticateUserByGoogle)
		r.Post("/auth/authenticate/manual", handlers.ApiAuthenticateUserByCredentials)
		r.Post("/auth/register", handlers.ApiRegisterUser)
		r.Post("/auth/resetpasswordmail", handlers.ApiResetPasswordMail)
		r.Get("/auth/verifyaccount", handlers.ApiVerifyAccount)
		r.Post("/auth/setpassword", handlers.ApiSetCredentials)
		r.Get("/auth/verifyPasswordExist/{email}", handlers.ApiCheckPasswordExist)

		// Stripe webhook (no JWT, uses Stripe signature verification)
		r.Post("/api/stripe_webhook_handler", handlers.StripeWebhookHandler)

		// JWT-protected routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.JWTMiddleware)

			r.Get("/auth/checkSession", handlers.ApiCheckSession)

			// Polizas / Scrapping
			r.Post("/scrapping/poliza", handlers.ApiPostPoliza)
			r.Get("/scrapping/details", handlers.ApiGetDetails)
			r.Get("/scrapping/polizas", handlers.ApiGetPolizas)
			r.Post("/scrapping/polizas", handlers.ApiPostPolizas)
			r.Get("/scrapping/polizas_ids", handlers.ApiGetPolizasIds)
			r.Get("/scrapping/poliza/{polizaNum}", handlers.ApiGetPoliza)
			r.Get("/polizas/birthdates", handlers.ApiGetBirthdates)

			// Poliza update
			r.Patch("/scrapping/poliza", handlers.ApiPatchPoliza)

			// Payments
			r.Patch("/payments/poliza", handlers.ApiSetPayment)

			// Stripe subscriptions
			r.Post("/api/create_suscription_payment", handlers.CreateStripeCheckoutSession)
			r.Get("/api/subscription_status", handlers.ApiGetSubscriptionStatus)
			r.Post("/api/cancel_subscription", handlers.ApiCancelSubscription)
		})
	})

	return r
}
