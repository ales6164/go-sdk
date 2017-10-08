package sdk

import (
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"github.com/gorilla/sessions"
	"github.com/gorilla/securecookie"
)

type SDK struct {
	Router *mux.Router
	*AppOptions
	middleware   *JWTMiddleware
	sessionStore *sessions.CookieStore
	installed bool
}

type AppOptions struct {
	SigningKey []byte
}

type Config struct {
	Key func(ctx context.Context) *datastore.Key
}

type MyServer struct {
	h *mux.Router
}

var signingKey []byte

func (s *MyServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, "+
				"X-Requested-With")
	}
	if req.Method == "OPTIONS" {
		return
	}
	s.h.ServeHTTP(w, req)
}

func NewApp(opt AppOptions) *SDK {
	a := &SDK{
		AppOptions: &opt,
	}

	if opt.SigningKey == nil {
		opt.SigningKey = securecookie.GenerateRandomKey(64)
	}

	signingKey = opt.SigningKey

	return a
}

func (a *SDK) EnableAdminDashboard() {
	AdminDashboard(a)
}

// Recommended path "/api/"
func (a *SDK) Serve(path string) {
	a.Router = mux.NewRouter().PathPrefix(path).Subrouter()

	a.middleware = AuthMiddleware(signingKey)

	a.HandleFunc("/profile", GetUserProfileHandler).Methods(http.MethodGet)
	a.HandleFunc("/profile", EditUserProfileHandler).Methods(http.MethodPut)
	a.HandleFunc("/auth/login", LoginHandler).Methods(http.MethodPost)
	a.HandleFunc("/auth/register", RegisterHandler).Methods(http.MethodPost)
	a.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		if ctx.err != nil {
			ctx.PrintError(w, ctx.err, http.StatusInternalServerError)
			return
		}
		if ctx.IsAuthenticated {
			ctx.Print(w, true)
			return
		}

		ctx.PrintError(w, ErrNotAuthenticated, http.StatusInternalServerError)
	}).Methods(http.MethodPost)

	// authorize and get user profile
	a.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithScopes(ScopeGet)
		if ctx.err != nil {
			ctx.PrintError(w, ctx.err, http.StatusInternalServerError)
			return
		}
		if ctx.IsAuthenticated {

			ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User, false)
			if err != nil {
				ctx.PrintError(w, err, http.StatusInternalServerError)
				return
			}

			d, err := ProfileEntity.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err, http.StatusInternalServerError)
				return
			}

			ctx.Print(w, d.Output())
			return
		}

		ctx.PrintError(w, ErrNotAuthenticated, http.StatusUnauthorized)
	}).Methods(http.MethodGet)


	http.Handle(path, &MyServer{a.Router})
}

func (a *SDK) Handle(path string, handler http.Handler) *mux.Route {
	return a.Router.Handle(path, a.middleware.Handler(handler))
}

func (a *SDK) HandleFunc(path string, handlerFunc func(w http.ResponseWriter, r *http.Request)) *mux.Route {
	return a.Router.Handle(path, a.middleware.Handler(http.HandlerFunc(handlerFunc)))
}

func (a *SDK) Handler(handlerFunc http.HandlerFunc) http.Handler {
	return a.middleware.Handler(http.Handler(handlerFunc))
}
