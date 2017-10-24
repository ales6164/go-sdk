package sdk

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
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

func (c *Context) NewUserToken(userKey string, userRoleKey string) error {
	var err error
	c.Token, err = newToken(userKey, userRoleKey)
	return err
}

func newToken(userKey string, userRoleKey string) (Token, error) {
	var tkn Token

	if len(userKey) == 0 || len(userRoleKey) == 0 {
		return tkn, ErrIllegalAction
	}

	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":  "api",
		"nbf":  time.Now().Add(-time.Minute).Unix(),
		"exp":  exp,
		"iat":  time.Now().Unix(),
		"iss":  "sdk",
		"sub":  userKey,
		"role": userRoleKey,
	})

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return tkn, err
	}

	return Token{signed, exp}, nil
}

/*
func (c *Context) NewAnonymousToken() (error) {
	var exp = time.Now().Add(time.Hour * 12).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":  "api",
		"nbf":  time.Now().Add(-time.Minute).Unix(),
		"exp":  exp,
		"iat":  time.Now().Unix(),
		"iss":  "sdk",
	})

	signed, err := token.SignedString(signingKey)
	if err != nil {
		return err
	}

	c.Token = Token{signed, exp}
	return nil
}

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
*/
