package sdk

import (
	"net/http"
)

type Context struct {
	r         *http.Request
}

func NewContext(r *http.Request) Context {
	return Context{
		r: r,
	}
}