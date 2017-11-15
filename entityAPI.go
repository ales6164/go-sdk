package sdk

import (
	"github.com/gorilla/mux"
	"google.golang.org/appengine/datastore"
	"net/http"
	"strconv"
	"errors"
)

var enabledEntityAPIs []*Entity

func (a *SDK) enableEntityAPI(e *Entity) {
	a.HandleFunc("/entity/"+e.Name, e.handleGetEntityInfo()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name+"/datatable", e.handleDataTable()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name+"/{encodedKey}", e.handleGet()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name, e.handleQuery()).Methods(http.MethodGet)
	a.HandleFunc("/"+e.Name, e.handleAdd()).Methods(http.MethodPost)
	a.HandleFunc("/"+e.Name+"/{encodedKey}", e.handleEdit()).Methods(http.MethodPost)

	// todo:
	// Gets single field value
	a.HandleFunc("/"+e.Name+"/{encodedKey}/{fieldName}", e.handleConstructPreviewURL()).Methods(http.MethodGet)
	// Updates single field value
	a.HandleFunc("/"+e.Name+"/{encodedKey}/{fieldName}", e.handleConstructPreviewURL()).Methods(http.MethodPost)

	enabledEntityAPIs = append(enabledEntityAPIs, e)
}

func (e *Entity) handleGetEntityInfo() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()
		ctx.Print(w, e)
	}
}

func (e *Entity) handleGetField() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()
		vars := mux.Vars(r)

		encodedKey := vars["encodedKey"]
		fieldName := vars["fieldName"]

		ctx, key, err := e.DecodeKey(ctx, encodedKey)
		if err != nil {
			ctx.PrintError(w, err, http.StatusBadRequest)
			return
		}

		dataHolder, err := e.GetField(ctx, key, fieldName)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, dataHolder.Output(ctx))
	}
}

func (e *Entity) handleConstructPreviewURL() func(w http.ResponseWriter, r *http.Request) {
	rt := mux.NewRouter().NewRoute().Path(e.Render.Path)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r).WithBody()

		holder, _ := emptyEntity.FromForm(ctx)

		var vars []string
		for key, val := range holder.input {
			vars = append(vars, key, val.(string))
		}

		url, err := rt.URL(vars...)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, url.String())
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

		var tableColumns []string
		var datatable []string
		if tc, ok := e.Meta["datatable"]; ok {
			if datatable, ok = tc.([]string); !ok {
				ctx.PrintError(w, errors.New("entity datatable meta definition should be of type string array"), http.StatusInternalServerError)
				return
			}
		} else {
			ctx.PrintError(w, errors.New("entity has no datatable meta definition"), http.StatusInternalServerError)
			return
		}
		if tc, ok := e.Meta["tableColumns"]; ok {
			if tableColumns, ok = tc.([]string); !ok {
				ctx.PrintError(w, errors.New("entity tableColumns meta definition should be of type string array"), http.StatusInternalServerError)
				return
			}
		} else {
			ctx.PrintError(w, errors.New("entity has no tableColumns meta definition"), http.StatusInternalServerError)
			return
		}

		if len(datatable) != len(tableColumns) {
			ctx.PrintError(w, errors.New("entity datatable and tableColumns definitions are not of the same length"), http.StatusInternalServerError)
			return
		}

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

		var data []interface{}
		for _, h := range dataHolder {
			var item = map[string]interface{}{}
			for i, fieldName := range datatable {
				item[tableColumns[i]] = h.Get(ctx, fieldName)
			}
			item["id"] = h.id
			data = append(data, item)
		}

		printOut(w, Result{
			"data": data,
		})
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

		filterField := q.Get("filter[field]")
		filterOp := q.Get("filter[op]")
		filterValue := q.Get("filter[value]")

		var filter EntityQueryFilter
		if len(filterField) > 0 && len(filterOp) > 0 {

			var op string
			switch filterOp {
			case "eq":
				op = "="
				break
			case "lt":
				op = "<"
				break
			case "le":
				op = "<="
				break
			case "gt":
				op = ">"
				break
			case "ge":
				op = ">="
				break
			}

			filter = EntityQueryFilter{
				Name:     filterField,
				Operator: op,
				Value:    filterValue,
			}
		}

		dataHolder, err := e.Query(ctx, sort, limit, filter)
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		var data []map[string]interface{}
		for _, h := range dataHolder {
			data = append(data, h.Output(ctx))
		}

		ctx.Print(w, map[string]interface{}{
			"data":  data,
			"count": len(data),
		})
	}
}
