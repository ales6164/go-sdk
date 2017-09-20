package sdk

import (
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"html/template"
	"net/http"
	"github.com/gorilla/sessions"
	gcontext"github.com/gorilla/context"
	"github.com/asaskevich/govalidator"
	"github.com/dgrijalva/jwt-go"
)

type SDK struct {
	*AppOptions
	Router       *mux.Router
	Middleware   *JWTMiddleware
	SessionStore *sessions.CookieStore
}

type AppOptions struct {
	ClientID        string
	ClientSecret    string
	TemplateFuncMap template.FuncMap
	SigningKey      []byte
	AppName         string
}

type Config struct {
	Key func(ctx context.Context) *datastore.Key
}

type MyServer struct {
	h *mux.Router
}

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

func NewApp(opt AppOptions) SDK {
	a := SDK{
		AppOptions:   &opt,
		Router:       mux.NewRouter(),
		Middleware:   NewAuthMiddleware(opt.SigningKey),
		SessionStore: sessions.NewCookieStore([]byte(opt.SigningKey)),
	}

	a.Router.Handle("/api/auth", a.Handle(a.CheckToken)).Methods(http.MethodPost)
	a.Router.HandleFunc("/api/auth/login", a.LoginHandler).Methods(http.MethodPost)
	a.Router.HandleFunc("/api/auth/register", a.RegisterHandler).Methods(http.MethodPost)

	return a
}

func (a *SDK) Serve(path string) {
	/*ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex) + "/admin"*/

	/*fs := http.FileServer(http.Dir("admin"))
	http.Handle("/admin/", http.StripPrefix("/admin/", fs))*/

	http.Handle(path, &MyServer{a.Router})
}


func (a *SDK) AddEntity(name string, properties []Property) *Entity {
	var e *Entity
	e = &Entity{name, properties, map[string]Property{}}

	for _, prop := range properties {
		e.PropertyMap[prop.Name] = prop
	}

	return e
}

func (a *SDK) Handle(handlerFunc http.HandlerFunc) http.Handler {
	return a.Middleware.Handler(http.Handler(handlerFunc))
}

func GetUser(r *http.Request) (string, bool) {
	user := gcontext.Get(r, "user")

	if user == nil {
		return "", false
	}

	var email string
	var claims jwt.MapClaims
	var ok bool
	token := user.(*jwt.Token)

	if claims, ok = token.Claims.(jwt.MapClaims); ok && token.Valid {
		if email, ok = claims["sub"].(string); !ok {
			return email, false
		}
		return email, true
	}

	return email, false
}
