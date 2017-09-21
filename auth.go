package sdk

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

func newAuthMiddleware(signingKey []byte) *JWTMiddleware {
	return New(Options{
		Extractor: FromFirst(
			FromAuthHeader,
			FromParameter("token"),
		),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		},
		SigningMethod:       jwt.SigningMethodHS256,
		CredentialsOptional: true,
	})
}

/*func (a *SDK) SetSession(w http.ResponseWriter, r *http.Request, id_token string) error {
	session, err := a.SessionStore.Get(r, "auth-session")
	if err != nil {
		return err
	}

	session.Values["id_token"] = id_token
	return session.Save(r, w)
}*/

type Token struct {
	ID      string `json:"id"`
	Expires int64  `json:"expires"`
}

func NewToken(usr string) (Token, error) {
	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"nbf": time.Now().Add(-time.Minute * 10).Unix(),
		"exp": exp,
		"iat": time.Now().Unix(),
		"iss": "sdk",
		"sub": usr,
	})

	signed, err := token.SignedString(signingKey)
	return Token{signed, exp}, err
}

/*func (a *SDK) AnonTokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := NewToken(uuid.New().String())
		if err != nil {
			printError(w, err, http.StatusInternalServerError)
			return
		}

		printData(w, token)
	})
}*/
