package sdk

import (
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
)

var enabledEntityAPIs []*Entity

func (a *SDK) enableEntityAPI(e *Entity) {
	a.HandleFunc("/entity/"+e.Name, e.handleGetEntityInfo()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name+"/{encodedKey}", e.handleGet()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name, e.handleQuery()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name+"/datatable", e.handleDataTable()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name, e.handleAdd()).Methods(http.MethodPost)
	a.HandleFunc("/"+e.Name+"/{encodedKey}", e.handleEdit()).Methods(http.MethodPost)

	enabledEntityAPIs = append(enabledEntityAPIs, e)
}

func (e *Entity) handleGetEntityInfo() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()
		ctx.Print(w, e)
	}
}

func (e *Entity) handleGet() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()
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

		ctx.Print(w, dataHolder.Output(ctx))
	}
}

func (e *Entity) handleAdd() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)

		holder, err := e.FromForm(ctx)
		if err != nil {
			ctx.PrintError(w, err, http.StatusBadRequest)
			return
		}

		var key *datastore.Key
		ctx, key = e.NewIncompleteKey(ctx)

		key, err = e.Add(ctx, key, holder)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, holder.Output(ctx))
	}
}

func (e *Entity) handleEdit() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)
		vars := mux.Vars(r)
		encodedKey := vars["encodedKey"]

		holder, err := e.FromForm(ctx)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		var key *datastore.Key
		if len(encodedKey) != 0 {
			ctx, key, err = e.DecodeKey(ctx, encodedKey)
			if err != nil {
				ctx.PrintError(w, err, http.StatusInternalServerError)
				return
			}
		} else {
			ctx, key = e.NewIncompleteKey(ctx)
		}

		key, err = e.Edit(ctx, key, holder)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, holder.Output(ctx))
	}
}

func (e *Entity) handleDataTable() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()

		q := r.URL.Query()

		sort := q.Get("sort")
		limitStr := q.Get("limit")
		var limit = 0
		if len(limitStr) != 0 {
			limit, _ = strconv.Atoi(limitStr)
		}

		dataHolder, err := e.Query(ctx, sort, limit)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		var data []map[string]interface{}
		for _, h := range dataHolder {
			data = append(data, h.Output(ctx))
		}


		//TODO
		/*write(w, ctx.Token, http.StatusOK, "", responseKey, response)

		ctx.Print(w, map[string]interface{}{
			"data":   data,
			"count":  len(data),
		})*/
	}
}

func (e *Entity) handleQuery() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()

		q := r.URL.Query()

		sort := q.Get("sort")
		limitStr := q.Get("limit")
		var limit = 0
		if len(limitStr) != 0 {
			limit, _ = strconv.Atoi(limitStr)
		}

		dataHolder, err := e.Query(ctx, sort, limit)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		var data []map[string]interface{}
		for _, h := range dataHolder {
			data = append(data, h.Output(ctx))
		}

		ctx.Print(w, map[string]interface{}{
			"data":   data,
			"count":  len(data),
		})
	}
}
