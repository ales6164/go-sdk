package sdk

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"time"
)

func AuthMiddleware(signingKey []byte) *JWTMiddleware {
	return New(MiddlewareOptions{
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

func (c *Context) NewUserToken(userKey string, userRole Role) error {
	var err error
	c.Token, err = newToken(userKey, userRole)
	return err
}

func newToken(userKey string, userRole Role) (Token, error) {
	var tkn Token

	if len(userKey) == 0 || len(userRole) == 0 {
		return tkn, ErrIllegalAction
	}

	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"nbf": time.Now().Add(-time.Minute).Unix(),
		"exp": exp,
		"iat": time.Now().Unix(),
		"iss": "sdk",
		"sub": userKey,
		"rol": userRole,
	})

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return tkn, err
	}

	return Token{signed, exp}, nil
}

// Deprecated
func (c *Context) NewAnonymousToken() error {
	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "api",
		"nbf": time.Now().Add(-time.Minute).Unix(),
		"exp": exp,
		"iat": time.Now().Unix(),
		"iss": "sdk",
	})

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return err
	}

	c.Token = Token{signed, exp}
	return nil
}

// Deprecated
func (a *SDK) AnonTokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		err := ctx.NewAnonymousToken()
		if err != nil {
			printError(w, err, http.StatusInternalServerError)
			return
		}
	})
}
