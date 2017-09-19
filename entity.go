package sdk

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/http"
	"google.golang.org/appengine/search"
)

type Entity struct {
	Name        string
	Properties  []Property
	PropertyMap map[string]Property
}

type Property struct {
	Name         string      `json:"name"`
	DefaultValue interface{} `json:"defaultValue"`
	NoIndex      bool        `json:"noIndex"`
	IsRequired   bool        `json:"isRequired"`
	IsParent     bool        `json:"isParent"`
	IsID         bool        `json:"isID"`
	IsArray      bool        `json:"isArray"`

	IsPrivate bool `json:"isPrivate"`

	Transform func(value interface{}) (interface{}, bool, error) `json:"-"`
	Validator func(value interface{}) bool                       `json:"-"`

	SearchField func(f search.Field, fun func(f search.Field)) `json:"-"`
	SearchFacet func(f search.Facet, fun func(f search.Facet)) `json:"-"`
}

type MultiValue interface {
}

const (
	ProductEntity string = "product"
)

var Entities = map[string]*Entity{}

func (e *Entity) AssembleFromRequest(r *http.Request) (map[string]interface{}, error) {
	var values []string
	for _, prop := range e.Properties {
		values = append(values, r.FormValue(prop.Name))
	}

	return e.AssembleFromValues(values...)
}

func (e *Entity) AssembleFromValues(values ...string) (map[string]interface{}, error) {
	var result = map[string]interface{}{}
	var err error

	if len(values) != len(e.Properties) {
		return result, errors.New("values length doesn't match properties length")
	}

	for i, prop := range e.Properties {
		var val = values[i]

		if len(val) == 0 {
			if prop.IsRequired {
				return result, errors.New("field '" + prop.Name + "' required")
			}
		}

		result[prop.Name] = val
	}

	return result, err
}

func (e *Entity) NewConnection(ctx context.Context) *Connection {
	return &Connection{Ctx: ctx, Entity: e, Name: e.Name}
}

func (e *Entity) Get(ctx context.Context, key *datastore.Key) (map[string]interface{}, error) {
	var result map[string]interface{}
	var err error

	var entData datastore.PropertyList
	if err = datastore.Get(ctx, key, &entData); err != nil {
		return result, err
	}

	result = map[string]interface{}{}
	for _, prop := range entData {
		result[prop.Name] = prop.Value
	}

	return result, err
}

func (e *Entity) NewQuery(ctx context.Context, offset int, limit int, ancestor *datastore.Key, filters ...string) (*datastore.Query, error) {
	var q *datastore.Query

	if len(filters)%2 != 0 {
		return q, errors.New("Query filter strings must be in pairs")
	}

	q = datastore.NewQuery(e.Name).Offset(offset).Limit(limit)

	if ancestor != nil {
		q = q.Ancestor(ancestor)
	}

	for i := 0; i < len(filters); i += 2 {
		q = q.Filter(filters[i], filters[i+1])
	}

	return q, nil
}

func (e *Entity) RunQuery(ctx context.Context, q *datastore.Query) ([]map[string]interface{}, error) {
	var data = []map[string]interface{}{}
	var err error

	it := q.Run(ctx)
	var c int
	for {
		var ps datastore.PropertyList
		if _, err = it.Next(&ps); err == datastore.Done {
			break
		} else if err != nil {
			return data, err
		}

		var d = map[string]interface{}{}
		for _, prop := range ps {
			d[prop.Name] = prop.Value
		}

		data = append(data, d)

		c++
	}

	return data, nil
}

func assembleEntity(e *Entity, ctx context.Context, unass map[string]interface{}) (*datastore.Key, datastore.PropertyList, *Stored, error) {
	var key *datastore.Key
	var props datastore.PropertyList
	var stored = new(Stored)
	var err error

	var keyNameValue string
	var parentKey *datastore.Key
	var data = map[string]interface{}{}
	for _, prop := range e.Properties {
		var val interface{} = unass[prop.Name]
		var isMultiple bool

		_, isString := val.(string)

		if val == nil || (isString && len(val.(string)) == 0) {
			if prop.IsRequired {
				return key, props, stored, errors.New("property '" + prop.Name + "' required")
			}
			val = prop.DefaultValue
		} else {
			if prop.Validator != nil && !prop.Validator(val) {
				return key, props, stored, errors.New("value '" + prop.Name + "' didn't pass validation")
			}

			if prop.Transform != nil {
				val, isMultiple, err = prop.Transform(val)
				if err != nil {
					return key, props, stored, err
				}
			}
		}

		if prop.IsParent {
			parentKey = datastore.NewKey(ctx, prop.Name, val.(string), 0, nil)
		}

		if prop.IsID {
			keyNameValue = val.(string)
		}

		props = append(props, datastore.Property{
			Name:     prop.Name,
			Value:    val,
			Multiple: isMultiple,
			NoIndex:  prop.NoIndex,
		})

		data[prop.Name] = val
	}

	stored.Data = append(stored.Data, data)

	if len(keyNameValue) == 0 {
		key = datastore.NewIncompleteKey(ctx, e.Name, parentKey)
	} else {
		key = datastore.NewKey(ctx, e.Name, keyNameValue, 0, parentKey)
	}

	return key, props, stored, err
}
