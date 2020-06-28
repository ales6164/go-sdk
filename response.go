package sdk

import (
	"encoding/json"
	"fmt"
	"google.golang.org/appengine/log"
	"net/http"
)

const responseKey = "result"

type Result map[string]interface{}

func printError(w http.ResponseWriter, err error, code int) {
	write(w, Token{}, code, err.Error(), responseKey, nil)
}

func printData(w http.ResponseWriter, response interface{}) {
	write(w, Token{}, http.StatusOK, "", responseKey, response)
}

func (c *Context) Print(w http.ResponseWriter, response interface{}) {
	write(w, c.Token, http.StatusOK, "", responseKey, response)
}

func (c *Context) PrintError(w http.ResponseWriter, err error, code int) {
	log.Errorf(c.Context, "Internal Error: %v", err)
	write(w, c.Token, code, err.Error(), responseKey, nil)
}

func write(w http.ResponseWriter, token Token, status int, message string, responseKey string, response interface{}) {
	var out = Result{
		"status": status,
	}

	if len(token.ID) != 0 {
		out["token"] = token
	}

	if len(message) > 0 {
		out["message"] = message
	}

	if response != nil {
		out[responseKey] = response
	}

	printOut(w, out)
}

func printOut(w http.ResponseWriter, out Result) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(out)
	if err != nil {
		fmt.Fprint(w, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": err.Error(),
		})
	}
}
