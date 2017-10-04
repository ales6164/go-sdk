package sdk

import (
	"net/http"
	"github.com/gorilla/mux"
)

func (e *Entity) EnableAPI(a *SDK) {
	a.HandleFunc("/api/entities/{kind}/{encodedKey}", e.handleGet).Methods(http.MethodGet)
	a.HandleFunc("/api/entities/{kind}", e.handleQuery).Methods(http.MethodGet)
}

func (e *Entity) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := NewContext(r)
	vars := mux.Vars(r)

	encodedKey := vars["encodedKey"]

	ctx, key, err := e.DecodeKey(ctx, encodedKey)
	if err != nil {
		ctx.PrintError(w, err, http.StatusBadRequest)
		return
	}

	dataHolder, err := e.Get(ctx, key)
	if err != nil {
		ctx.PrintError(w, err, http.StatusInternalServerError)
		return
	}

	ctx.Print(w, dataHolder.Output())
}

func (e *Entity) handleQuery(w http.ResponseWriter, r *http.Request) {
	// TODO
}