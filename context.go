package sdk

import (
	"github.com/dgrijalva/jwt-go"
	gctx "github.com/gorilla/context"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"io/ioutil"
	"net/http"
	"time"
)

type Context struct {
	r      *http.Request
	scopes map[Scope]bool
	err    error

	Context context.Context

	User string
	Role string

	IsAuthenticated bool
	Token           Token

	body []byte
}

func NewContext(r *http.Request) Context {
	isAuthenticated, userRole, userKey, renewedToken, err := getUser(r)
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	if len(userRole) == 0 {
		userRole = "guest"
	}

	return Context{
		r:               r,
		Context:         appengine.NewContext(r),
		IsAuthenticated: isAuthenticated,
		Role:            userRole,
		User:            userKey,
		Token:           renewedToken,
		err:             err,
		body:            body,
	}
}

// return true if userKey matches with userKey in token
func (c Context) UserMatches(userKey interface{}) bool {
	if userKeyString, ok := userKey.(string); ok {
		return userKeyString == c.User
	}
	return false
}

func getUser(r *http.Request) (bool, string, string, Token, error) {
	var isAuthenticated bool
	var userRoleKey string
	var userKey string
	var renewedToken Token
	var err error

	tkn := gctx.Get(r, "user")

	if tkn != nil {
		token := tkn.(*jwt.Token)

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			err = claims.Valid()
			if err == nil {
				if username, ok := claims["sub"].(string); ok {
					if userRoleKey, ok := claims["role"].(string); ok {
						return true, userRoleKey, username, renewedToken, err
					}
				}
				return isAuthenticated, userRoleKey, userKey, renewedToken, ErrIllegalAction
			} else if exp, ok := claims["exp"].(float64); ok {
				// check if it's less than a week old
				if time.Now().Unix()-int64(exp) < time.Now().Add(time.Hour*24*7).Unix() {
					if userKey, ok := claims["sub"].(string); ok {
						if userRoleKey, ok := claims["role"].(string); ok {
							renewedToken, err = newToken(userKey, userRoleKey)
							if err != nil {
								return isAuthenticated, userRoleKey, userKey, renewedToken, err
							}
							return true, userRoleKey, userKey, renewedToken, err
						}
					}
				}
			}
		}
	}

	return isAuthenticated, userRoleKey, userKey, renewedToken, err
}
