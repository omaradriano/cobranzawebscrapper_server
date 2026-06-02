package configs

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/handlers"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
)

type Env struct {
	DbClient *sql.DB
}

var routes = []route{
	// newRoute("PATCH", "/v1/cobranzaItem", handlers.ApiPatchCobranzaItem),
	newRoute("PATCH", "/v1/payments/poliza", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiSetPayment)).ServeHTTP(w, r)
	}),

	/**
	 * Subir una poliza
	 */
	newRoute("POST", "/v1/scrapping/poliza", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiPostPoliza)).ServeHTTP(w, r)
	}),
	/**
	 * Detalles en la pantalla popup
		* NOTA: Estos detalles son para obtener las polizas:
		*  - Registradas
		*  - Activas
		*  - Por vencer
		*  - Tienen cobertura
		*  - Entre otras, verificar xd
	*/
	newRoute("GET", "/v1/scrapping/details", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetDetails)).ServeHTTP(w, r)
	}),
	/**
	 * Obtener todas las polizas
		*
		* http://server/v1/scrapping/polizas?status=
	*/
	newRoute("GET", "/v1/scrapping/polizas", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetPolizas)).ServeHTTP(w, r)
	}),
	/**
	 * Subir todas las polizas
	 */
	newRoute("POST", "/v1/scrapping/polizas", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiPostPolizas)).ServeHTTP(w, r)
	}),
	/**
	 * Obtener solo los numeros de poliza
		* NOTA: esta parte se usa para obtener la lista de los num_poliza para poder comparar
		* con las polizas de la extensión, para no tener que correr todo el scrapping desde cero
	*/
	newRoute("GET", "/v1/scrapping/polizas_ids", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetPolizasIds)).ServeHTTP(w, r)
	}),
	/**
	 * Solicitar una poliza
	 */
	newRoute("GET", "/v1/scrapping/poliza/([^/]+)", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetPoliza)).ServeHTTP(w, r)
	}),
	/**
	 * Obtener fechas de siguientes cumpleanos
	 */
	newRoute("GET", "/v1/polizas/birthdates", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetBirthdates)).ServeHTTP(w, r)
	}),

	// Payment -----------------------------
	/**
	 * Generacion de pago de Stripe
	 */
	newRoute("POST", "/v1/api/create_suscription_payment", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.CreateStripeCheckoutSession)).ServeHTTP(w, r)
	}),

	newRoute("POST", "/v1/api/stripe_webhook_handler", handlers.StripeWebhookHandler),

	newRoute("GET", "/v1/api/subscription_status", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiGetSubscriptionStatus)).ServeHTTP(w, r)
	}),

	// Authentication ----------------------------
	//
	/** Llamada a autenticación con google */
	newRoute("POST", "/v1/auth/authenticate/google", handlers.ApiAuthenticateUserByGoogle),

	/** Autenticación con credenciales manuales (usuario y contraseña) */
	newRoute("POST", "/v1/auth/authenticate/manual", handlers.ApiAuthenticateUserByCredentials),

	/** Registro de usuario (WebApp) */
	newRoute("POST", "/v1/auth/register", handlers.ApiRegisterUser),

	/** Envio de correo para reiniciar contraseña */
	newRoute("POST", "/v1/auth/resetpasswordmail", handlers.ApiResetPasswordMail),

	/** Verificación de cuenta con el token generado en /v1/auth/resetpasswordmail */
	newRoute("GET", "/v1/auth/verifyaccount", handlers.ApiVerifyAccount),

	/** Verificación de existencia de una sesión con JWT (En este caso lo guardamos en la sesion de la extensión) */
	newRoute("GET", "/v1/auth/checkSession", func(w http.ResponseWriter, r *http.Request) {
		middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiCheckSession)).ServeHTTP(w, r)
	}),
	/** Actualización de los datos de una poliza */
	// newRoute("PATCH", "/v1/scrapping/poliza", func(w http.ResponseWriter, r *http.Request) {
	// 	middlewares.JWTMiddleware(http.HandlerFunc(handlers.ApiPatchPoliza)).ServeHTTP(w, r)
	// }),

	/** Reinicio de contraseña (WebApp) */
	newRoute("POST", "/v1/auth/setpassword", handlers.ApiSetCredentials),

	/**
	 * Comprueba que el usuario cuente con una contraseña
		* NOTA: este flujo verifica que el usuario tenga una contraseña en caso de tener un registro
		* o inicio de sesión con google. En caso de no tener una contraseña se hace una redirección
		* a la vista para establecer una (La redirección o apertura es desde la extensión)
	*/
	newRoute("GET", "/v1/auth/verifyPasswordExist/([^/]+)", handlers.ApiCheckPasswordExist),
}

func newRoute(method, pattern string, handler http.HandlerFunc) route {
	return route{method, regexp.MustCompile("^" + pattern + "$"), handler}
}

type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func Serve(w http.ResponseWriter, r *http.Request) {
	var allow []string
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			if r.Method != route.method {
				allow = append(allow, route.method)
				continue
			}
			ctx := context.WithValue(r.Context(), internal.CtxKey{}, matches[1:])
			route.handler(w, r.WithContext(ctx))
			return
		}
	}
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, r)
}
