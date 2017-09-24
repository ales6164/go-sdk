package sdk

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"fmt"
)

const secureSessionName = "auth-session"

var session *sessions.CookieStore
var subRouter *mux.Router
var adminMiddleware *JWTMiddleware

func AdminDashboard(a *SDK) {
	subRouter = a.Router.PathPrefix("/admin").Subrouter()
	adminMiddleware = AdminMiddleware(a.SigningKey)

	subRouter.Handle("/", adminMiddleware.Handler(http.HandlerFunc(AdminDashboardHandler)))
	subRouter.Handle("/sign-in", http.HandlerFunc(AdminLoginHandler))
}

func AdminMiddleware(signingKey []byte) *JWTMiddleware {
	session = sessions.NewCookieStore(signingKey)

	return New(Options{
		Extractor: FromFirst(
			/*FromAuthHeader,*/
			FromSession(secureSessionName),
		),
		RedirectOnError: "/admin/sign-in",
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		},
		SigningMethod:       jwt.SigningMethodHS256,
		CredentialsOptional: true,
	})
}

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Dashboard")
}

func AdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Sign In")
}
