package sdk

import (
	"net/http"
	"github.com/dgrijalva/jwt-go"
	gctx"github.com/gorilla/context"
	"time"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
)

type Context struct {
	r *http.Request

	ctx context.Context

	username        string
	namespace       string
	isAuthenticated bool
	token           Token
	err             error
}

func NewContext(r *http.Request) Context {
	isAuthenticated, username, renewedToken, err := getUser(r)
	return Context{
		r:               r,
		ctx:             appengine.NewContext(r),
		isAuthenticated: isAuthenticated,
		username:        username,
		token:           renewedToken,
		err:             err,
	}
}

func getUser(r *http.Request) (bool, string, Token, error) {
	var isAuthenticated bool
	var username string
	var renewedToken Token
	var err error

	user := gctx.Get(r, "user")

	if user != nil {
		token := user.(*jwt.Token)

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			err = claims.Valid()
			if err == nil {
				if username, ok := claims["sub"].(string); ok {
					return true, username, renewedToken, err
				}
			} else if err == jwt.ValidationError(jwt.ValidationErrorExpired) {
				if exp, ok := claims["exp"].(int64); ok {
					// check if it's less than a week old
					if time.Now().Unix()-exp < time.Now().Add(time.Hour * 24 * 7).Unix() {
						if username, ok := claims["sub"].(string); ok {
							renewedToken, err = NewToken(username)
							if err != nil {
								return isAuthenticated, username, renewedToken, err
							}
							return true, username, renewedToken, err
						}
					}
				}
			}
		}
	}

	return isAuthenticated, username, renewedToken, err
}
