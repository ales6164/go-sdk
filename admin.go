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

const adminDir = "admin/dist"

func AdminDashboard(a *SDK) {
	/*fs := http.FileServer(http.Dir(adminDir))

	a.Router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if a.installed {
			fs.ServeHTTP(w, r)
			//http.Redirect(w, r, "/admin", http.StatusPermanentRedirect)
			return
		}

		ctx := appengine.NewContext(r)
		client := urlfetch.Client(ctx)
		resp, err := client.Get("https://storage.googleapis.com/tiskdaril.appspot.com/dist.zip")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var fileName = "_temp-admin.zip"

		defer resp.Body.Close()
		out, err := os.Create(fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = Unzip(fileName, adminDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		a.installed = true

		fs.ServeHTTP(w, r)
	})*/

	//a.Router.PathPrefix("/admin").Handler(fs)
	//subRouter.Handle("/sign-in", http.HandlerFunc(AdminLoginHandler))
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

func AdminLoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Sign In")
}
