package sdk

import (
	"net/http"
	"github.com/gorilla/mux"
)

func (e *Entity) EnableAPI(a *SDK, fieldPosition []string) {

	a.HandleFunc("/api/entities/"+e.Name+"/{encodedKey}", e.handleGet(fieldPosition)).Methods(http.MethodGet)
	a.HandleFunc("/api/entities/"+e.Name, e.handleQuery(fieldPosition)).Methods(http.MethodGet)
}

func (e *Entity) handleGet(fieldPosition []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ctx.Print(w, map[string]interface{}{
			"fields": fieldPosition,
			"data": dataHolder.Output(),
		})
	}
}

func (e *Entity) handleQuery(fieldPosition []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithScopes(ScopeGet)

		dataHolders, err := e.Query(ctx, "", "", 100)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		var data []map[string]interface{}
		for _, dataHolder := range dataHolders {
			data = append(data, dataHolder.Output())
		}

		ctx.Print(w, map[string]interface{}{
			"fields": fieldPosition,
			"data": data,
		})
	}
}
