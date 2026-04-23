package configs

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/handlers"
)

type Env struct {
	DbClient *sql.DB
}

var routes = []route{
	newRoute("GET", "/v1/cobranzasItems/([^/]+)", handlers.ApiGetCobranzasItems),
	newRoute("POST", "/v1/cobranzaItem", handlers.ApiPostCobranzaItem),
	// newRoute("PATCH", "/v1/cobranzaItem", handlers.ApiPatchCobranzaItem),
	newRoute("PATCH", "/v1/cobranzaItemPayment", handlers.ApiPatchItemPayment),
	newRoute("POST", "/v1/cobranzaAllItems", handlers.ApiPostCobranzaAllItems),
	// newRoute("PATCH", "/v1/cobranzaFewItems", handlers.ApiPatchFewCobranzas),

	newRoute("POST", "/v1/auth/authenticate/google", handlers.ApiAuthenticateUserByGoogle),
	newRoute("POST", "/v1/auth/authenticate/manual", handlers.ApiAuthenticateUserByCredentials),
	newRoute("POST", "/v1/auth/register", handlers.ApiRegisterUser),
	newRoute("POST", "/v1/auth/resetpasswordmail", handlers.ApiResetPasswordMail),
	newRoute("GET", "/v1/auth/verifyaccount", handlers.ApiVerifyAccount),
	newRoute("GET", "/v1/auth/checkSession", handlers.ApiCheckSession),
	newRoute("POST", "/v1/auth/setpassword", handlers.ApiSetPassword),
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
