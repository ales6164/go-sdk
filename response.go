package sdk

import (
	"encoding/json"
	"net/http"
	"fmt"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func PrintError(w http.ResponseWriter, err error, code int) {
	printError(w, err, code)
}

func PrintData(w http.ResponseWriter, data interface{}) {
	printData(w, data)
}

func printError(w http.ResponseWriter, err error, code int) {
	write(w, Response{
		Code:    code,
		Message: err.Error(),
	}, code)
}

func printData(w http.ResponseWriter, data interface{}) {
	write(w, Response{
		Code:   http.StatusOK,
		Result: data,
	}, http.StatusOK)
}

func write(w http.ResponseWriter, result Response, status int) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		fmt.Fprint(w, Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}
	w.WriteHeader(http.StatusOK)
}
