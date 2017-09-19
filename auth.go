package sdk

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"net/http"
	"time"
)

func NewAuthMiddleware(signingKey []byte) *JWTMiddleware {
	return New(Options{
		Extractor: FromFirst(
			FromAuthHeader,
			FromParameter("token"),
		),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
}

func (a *SDK) SetSession(w http.ResponseWriter, r *http.Request, id_token string) error {
	session, err := a.SessionStore.Get(r, "auth-session")
	if err != nil {
		return err
	}

	session.Values["id_token"] = id_token
	return session.Save(r, w)
}

func (a *SDK) NewToken(usr string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"exp": time.Now().Add(time.Hour * 12).Unix(),
		"iat": time.Now().Unix(),
		"iss": a.AppName,
		"sub": usr,
	})

	return token.SignedString(a.SigningKey)
}

func (a *SDK) AnonTokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := a.NewToken(uuid.New().String())
		if err != nil {
			printError(w, err, http.StatusInternalServerError)
			return
		}

		printData(w, token)
	})
}
