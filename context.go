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
	r      *http.Request
	scopes map[Scope]bool
	err    error

	Context         context.Context
	User            string
	Namespace       string
	IsAuthenticated bool
	Token           Token
}

func NewContext(r *http.Request) Context {
	isAuthenticated, namespace, username, renewedToken, err := getUser(r)
	return Context{
		r:               r,
		Context:         appengine.NewContext(r),
		IsAuthenticated: isAuthenticated,
		Namespace:       namespace,
		User:            username,
		Token:           renewedToken,
		err:             err,
	}
}

func (c Context) HasScope(scope Scope) bool {
	return c.scopes[scope]
}

func (c Context) WithScopes(scopes ...Scope) Context {
	c.scopes = map[Scope]bool{}
	for _, scope := range scopes {
		c.scopes[scope] = true
	}
	return c
}

func (c Context) WithNamespace()  {
	if c.IsAuthenticated && len(c.Namespace) != 0 {
		c.Context, c.err = appengine.Namespace(c.Context, c.Namespace)
	} else {
		c.err = ErrNotAuthenticated
	}
}

func getUser(r *http.Request) (bool, string, string, Token, error) {
	var isAuthenticated bool
	var namespace string
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
					if namespace, ok := claims["namespace"].(string); ok {
						return true, namespace, username, renewedToken, err
					}
				}
				return isAuthenticated, namespace, username, renewedToken, ErrIllegalAction
			} else if err == jwt.ValidationError(jwt.ValidationErrorExpired) {
				if exp, ok := claims["exp"].(int64); ok {
					// check if it's less than a week old
					if time.Now().Unix()-exp < time.Now().Add(time.Hour * 24 * 7).Unix() {
						if username, ok := claims["sub"].(string); ok {
							if namespace, ok := claims["namespace"].(string); ok {
								renewedToken, err = NewToken(namespace, username)
								if err != nil {
									return isAuthenticated, namespace, username, renewedToken, err
								}
								return true, namespace, username, renewedToken, err
							}
						}
					}
				}
			}
		}
	}

	return isAuthenticated, namespace, username, renewedToken, err
}
