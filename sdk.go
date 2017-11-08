package sdk

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type SDK struct {
	Router *mux.Router
	*AppOptions
	middleware   *JWTMiddleware
	sessionStore *sessions.CookieStore
	installed    bool
}

type AppOptions struct {
	SigningKey []byte
	AdminEmail string
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

const apiPath = "/api/"

type API struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

func NewApp(opt AppOptions) *SDK {
	a := &SDK{
		AppOptions: &opt,
	}

	if opt.SigningKey == nil {
		opt.SigningKey = securecookie.GenerateRandomKey(64)
	}

	signingKey = opt.SigningKey

	a.Router = mux.NewRouter().PathPrefix(apiPath).Subrouter()
	a.middleware = AuthMiddleware(signingKey)
	http.Handle(apiPath, &MyServer{a.Router})

	// handler returns enabled apis
	a.HandleFunc("/entities", func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		ctx.Print(w, enabledEntityAPIs)
	})

	// client handler
	if _, err := clientIdSecret.init(); err != nil {
		panic(err)
	}
	a.Handle("/auth/client", http.HandlerFunc(NewClientRequest(a)))
	a.Handle("/auth/client/issue-token", http.HandlerFunc(IssueClientToken))

	return a
}

/*func (a *SDK) EnableAdminDashboard() {
	AdminDashboard(a)
}*/

// Recommended path "/api/"
func (a *SDK) EnableAuthAPI() {
	if _, err := lostPasswordRequest.init(); err != nil {
		panic(err)
	}
	if _, err := userEntity.init(); err != nil {
		panic(err)
	}
	if _, err := ProfileEntity.init(); err != nil {
		panic(err)
	}

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
		ctx := NewContext(r).WithScopes(ScopeRead)
		if ctx.err != nil {
			ctx.PrintError(w, ctx.err, http.StatusInternalServerError)
			return
		}
		if ctx.IsAuthenticated {

			ctx, key, err := ProfileEntity.NewKey(ctx, ctx.User)
			if err != nil {
				ctx.PrintError(w, err, http.StatusInternalServerError)
				return
			}

			d, err := ProfileEntity.Get(ctx, key)
			if err != nil {
				ctx.PrintError(w, err, http.StatusInternalServerError)
				return
			}

			ctx.Print(w, d.Output(ctx))
			return
		}

		ctx.PrintError(w, ErrNotAuthenticated, http.StatusUnauthorized)
	}).Methods(http.MethodGet)
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

//var DefaultDataHolder = NewEntity("DefaultDataHolder", []*Field{})
