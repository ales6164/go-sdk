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
	*AppOptions
	Router       *mux.Router
	middleware   *JWTMiddleware
	sessionStore *sessions.CookieStore
}

type AppOptions struct {
	SigningKey         []byte
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
		Router:     mux.NewRouter(),
	}

	if opt.SigningKey == nil {
		opt.SigningKey = securecookie.GenerateRandomKey(64)
	}

	signingKey = opt.SigningKey
	a.middleware = AuthMiddleware(signingKey)

	a.HandleFunc("/api/profile", GetUserProfileHandler).Methods(http.MethodGet)
	a.HandleFunc("/api/profile", EditUserProfileHandler).Methods(http.MethodPut)
	a.HandleFunc("/api/auth/login", LoginHandler).Methods(http.MethodPost)
	a.HandleFunc("/api/auth/register", RegisterHandler).Methods(http.MethodPost)
	a.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		if ctx.err != nil {
			ctx.PrintError(w, ctx.err, http.StatusInternalServerError)
			return
		}
		if ctx.IsAuthenticated {
			ctx.Print(w, true)
			return
		}

		ctx.PrintError(w, ErrNotAuthenticated, http.StatusNetworkAuthenticationRequired)
	}).Methods(http.MethodPost)

	// authorize and get user profile
	a.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
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

	return a
}

func (a *SDK) EnableAdminDashboard() {
	AdminDashboard(a)
}

func (a *SDK) Serve(path string) {
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
