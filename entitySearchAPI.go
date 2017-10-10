package sdk

import (
	"net/http"
	"google.golang.org/appengine/search"
	"golang.org/x/net/context"
	"net/url"
	"strconv"
)

func (a *SDK) EnableEntitySearchAPI(e *Entity, index *DocumentDefinition, fieldPosition []string) {
	a.HandleFunc("/"+e.Name+"/search", e.handleSearch(index, fieldPosition)).Methods(http.MethodGet)
}


func (e *Entity) handleSearch(dd *DocumentDefinition, fieldPosition []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := NewContext(r)

		results, err := indexQuery(ctx.Context, dd, r.URL.Query())
		if err != nil {
			ctx.PrintError(w, err, http.StatusInternalServerError)
			return
		}

		ctx.Print(w, map[string]interface{}{
			"fields": fieldPosition,
			"data":   results,
			"count":  len(results),
		})
	}
}

func indexQuery(ctx context.Context, dd *DocumentDefinition, query url.Values) ([]interface{}, error) {
	var data []interface{}

	index, err := search.Open(dd.Name)
	if err != nil {
		return data, err
	}

	var searchString = query.Get("q")
	var sort = query.Get("sort")
	var limit_str = query.Get("limit")
	var offset_str = query.Get("offset")

	var sortExpr []search.SortExpression
	if len(sort) != 0 {
		var desc bool
		if sort[:1] == "-" {
			sort = sort[1:]
			desc = true
		}
		sortExpr = append(sortExpr, search.SortExpression{Expr: sort, Reverse: !desc})
	}

	var limit = 25 // default limit
	if len(limit_str) != 0 {
		limit, err = strconv.Atoi(limit_str)
		if err != nil {
			return data, err
		}
	}

	var offset = 0 // default offset
	if len(offset_str) != 0 {
		offset, err = strconv.Atoi(offset_str)
		if err != nil {
			return data, err
		}
	}

	var it = index.Search(ctx, searchString, &search.SearchOptions{
		Sort: &search.SortOptions{
			Expressions: sortExpr,
		},
		Limit:  limit,
		Offset: offset,
	})

	for {
		var doc Document
		_, err := it.Next(&doc)
		if err == search.Done {
			break
		}
		if err != nil {
			return data, err
		}
		var docData = map[string]interface{}{}
		for _, field := range doc.Fields {

			if val, ok := docData[field.Name]; ok {
				// check if it's not an array already and create an array with old value
				if _, ok := val.([]interface{}); !ok {
					docData[field.Name] = []interface{}{val}
				}

				docData[field.Name] = append(docData[field.Name].([]interface{}), field.Value)
			} else {
				docData[field.Name] = field.Value
			}
		}
		data = append(data, docData)
	}

	return data, nil
}
