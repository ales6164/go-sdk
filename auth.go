package sdk

import (
	"github.com/dgrijalva/jwt-go"
	"time"
	"errors"
	"net/http"
	"github.com/google/uuid"
)

func AuthMiddleware(signingKey []byte) *JWTMiddleware {
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

type Token struct {
	ID      string `json:"id"`
	Expires int64  `json:"expires"`
}

var (
	ErrIllegalAction = errors.New("illegal action")
)

func NewToken(ns string, usr string) (Token, error) {
	if len(ns) == 0 && len(usr) == 0 {
		return Token{}, ErrIllegalAction
	}

	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":       "api",
		"nbf":       time.Now().Add(-time.Minute * 10).Unix(),
		"exp":       exp,
		"iat":       time.Now().Unix(),
		"iss":       "sdk",
		"sub":       usr,
		"namespace": ns,
	})

	signed, err := token.SignedString(signingKey)

	var t = Token{signed, exp}

	return t, err
}

/*func SetSecureSession(id_token string, w http.ResponseWriter, r *http.Request) error {
	session, err := session.Get(r, secureSessionName)
	if err != nil {
		return err
	}
	session.Values["id_token"] = id_token
	return session.Save(r, w)
}*/

func (a *SDK) AnonTokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := NewToken("", uuid.New().String())
		if err != nil {
			printError(w, err, http.StatusInternalServerError)
			return
		}

		printData(w, token)
	})
}
