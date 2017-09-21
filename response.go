package sdk

import (
	"encoding/json"
	"net/http"
	"fmt"
)

func printError(w http.ResponseWriter, err error, code int) {
	write(w, Token{}, code, err.Error(), nil)
}

func printData(w http.ResponseWriter, response interface{}) {
	write(w, Token{}, http.StatusOK, "", response)
}

func (c *Context) Print(w http.ResponseWriter, response interface{}) {
	write(w, c.Token, http.StatusOK, "", response)
}

func (c *Context) PrintError(w http.ResponseWriter, err error, code int) {
	write(w, c.Token, code, err.Error(), nil)
}

func write(w http.ResponseWriter, token Token, status int, message string, response interface{}) {
	var out = map[string]interface{}{
		"status": status,
	}

	if len(token.ID) != 0 {
		out["token"] = token
	}

	if len(message) > 0 {
		out["message"] = message
	}

	if response != nil {
		out["result"] = response
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(out)
	if err != nil {
		fmt.Fprint(w, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": err.Error(),
		})
	}
}
