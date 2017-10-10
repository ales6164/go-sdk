package sdk

import (
	"google.golang.org/appengine/datastore"
	"regexp"
	"fmt"
	"errors"
	"github.com/asaskevich/govalidator"
	"reflect"
	"google.golang.org/appengine/delay"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"encoding/json"
)

type Entity struct {
	KeepInCache bool //todo: if true loads all values and holds them in cache - good for categories, translations, ...

	Name        string
	Fields      map[string]*Field
	fields      []*Field

	preparedData map[*Field]func(*Field) interface{}

	requiredFields []*Field

	indexes map[string]*DocumentDefinition

	// listeners
	OnAfterRead func(h *EntityDataHolder) (error)
}

func NewEntity(name string, fields []*Field) *Entity {
	var e = new(Entity)
	e.Name = name
	e.fields = fields
	e.indexes = map[string]*DocumentDefinition{}

	e.preparedData = map[*Field]func(*Field) interface{}{}

	e.Fields = map[string]*Field{}
	for _, field := range fields {
		if len(field.Name) == 0 {
			panic(errors.New("field name can't be empty"))
		}

		if field.Name == "_id" || field.Name == "id" {
			panic(errors.New("field name _id/id is reserved and can't be used"))
		}

		if len(field.GroupName) != 0 {
			if !govalidator.IsAlpha(field.GroupName) {
				panic(errors.New("field group name contains non-alpha characters"))
			}
			field.datastoreFieldName = field.GroupName + "[" + field.Name + "]"
		} else {
			field.datastoreFieldName = field.Name
		}

		e.Fields[field.datastoreFieldName] = field

		if field.IsRequired {
			e.requiredFields = append(e.requiredFields, field)
		}

		if field.DefaultValue != nil {
			e.preparedData[field] = func(f *Field) interface{} {
				return f.DefaultValue
			}
		}

		if field.ValueFunc != nil {
			e.preparedData[field] = func(f *Field) interface{} {
				return f.ValueFunc()
			}
		}

		if len(field.ValidateRgx) > 0 {
			field.fieldFunc = append(field.fieldFunc, func(c *ValueContext, v interface{}) (interface{}, error) {
				if c.Trust == High {
					return v, nil
				}

				var matched bool
				var err error

				switch val := v.(type) {
				case string:
					matched, err = regexp.Match(field.ValidateRgx, []byte(val))
					break
				default:
					return v, fmt.Errorf(ErrFieldValueNotValid, c.Field.Name)
				}

				if err != nil {
					return nil, err
				}
				if matched {
					return v, nil
				}

				return v, fmt.Errorf(ErrFieldValueNotValid, c.Field.Name)
			})
		}

		if field.Validator != nil {
			field.fieldFunc = append(field.fieldFunc, func(c *ValueContext, v interface{}) (interface{}, error) {
				if c.Trust == High {
					return v, nil
				}

				ok := c.Field.Validator(v)
				if ok {
					return v, nil
				}
				return v, fmt.Errorf(ErrFieldValueNotValid, c.Field.Name)
			})
		}

		if field.TransformFunc != nil {
			field.fieldFunc = append(field.fieldFunc, field.TransformFunc)
		}
	}

	return e
}

/**
Adds index document definition and subscribes it to data changes
 */
func (e *Entity) AddIndex(dd *DocumentDefinition) {
	e.indexes[dd.Name] = dd
}

var putToIndex = delay.Func(RandStringBytesMaskImprSrc(16), func(ctx context.Context, dd DocumentDefinition, id string, data Data) {
	dd.Put(ctx, id, flatOutput(id, data))
})
var removeFromIndex = delay.Func(RandStringBytesMaskImprSrc(16), func(ctx context.Context, dd DocumentDefinition) {
	// do something expensive!
})

func (e *Entity) PutToIndexes(ctx context.Context, id string, data *EntityDataHolder) {
	for _, dd := range e.indexes {
		err := putToIndex.Call(ctx, *dd, id, data.data)
		if err != nil {
			log.Errorf(ctx, "%v", err.Error())
		}
	}
}
func (e *Entity) RemoveFromIndexes(ctx context.Context) {
	for _, dd := range e.indexes {
		removeFromIndex.Call(ctx, *dd)
	}
}

func (e *Entity) New() *EntityDataHolder {
	var dataHolder = &EntityDataHolder{
		Entity: e,
		data:   Data{},
		input:  map[string]interface{}{},
		isNew:  true,
	}

	// copy prepared values
	for field, fun := range e.preparedData {
		dataHolder.data[field] = fun(field)
	}

	return dataHolder
}

var (
	ErrKeyNameIdNil         = errors.New("key nameId is nil")
	ErrKeyNameIdInvalidType = errors.New("key nameId invalid type (only string/int64)")
)

func (e *Entity) DecodeKey(c Context, encodedKey string) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	key, err = datastore.DecodeKey(encodedKey)
	if err != nil {
		return c, key, err
	}

	if len(key.Namespace()) != 0 {
		c.WithNamespace()
	}

	return c, key, err
}

func (e *Entity) NewIncompleteKey(c Context, withNamespace bool) (Context, *datastore.Key) {
	var key *datastore.Key

	if withNamespace {
		c.WithNamespace()
	}

	key = datastore.NewIncompleteKey(c.Context, e.Name, nil)

	return c, key
}

// Gets appengine context and datastore key with optional namespace. It doesn't fail if request is not authenticated.
func (e *Entity) NewKey(c Context, nameId interface{}, withNamespace bool) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	if nameId == nil {
		return c, key, ErrKeyNameIdNil
	}

	if withNamespace {
		c.WithNamespace()
	}

	if stringId, ok := nameId.(string); ok {
		key = datastore.NewKey(c.Context, e.Name, stringId, 0, nil)
	} else if intId, ok := nameId.(int64); ok {
		key = datastore.NewKey(c.Context, e.Name, "", intId, nil)
	} else {
		return c, key, ErrKeyNameIdInvalidType
	}

	return c, key, err
}

func (e *Entity) FromForm(c Context) (*EntityDataHolder, error) {
	var h *EntityDataHolder

	return e.FromBody(c)

	// todo: fix this
	c.r.FormValue("a")

	var err error
	if err = c.r.ParseForm(); err != nil {
		return h, err
	}

	if len(c.r.Form) != 0 {
		h = e.New()
		for name, values := range c.r.Form {

			// remove '[]' from fieldName if it's an array
			if len(name) > 2 && name[len(name)-2:] == "[]" {
				name = name[:len(name)-2]
			}

			for _, v := range values {
				err = h.appendValue(name, v, Low)
				if err != nil {
					return h, err
				}
			}
		}
	} else {
		return e.FromBody(c)
	}

	return h, err
}

func (e *Entity) FromBody(c Context) (*EntityDataHolder, error) {
	var h *EntityDataHolder
	var err error

	if len(c.body) == 0 {
		return e.FromMap(c, map[string]interface{}{})
	}

	var t map[string]interface{}
	err = json.Unmarshal(c.body, &t)
	if err != nil {
		return h, err
	}

	return e.FromMap(c, t)
}

func (e *Entity) FromMap(c Context, m map[string]interface{}) (*EntityDataHolder, error) {
	var h = e.New()
	var err error

	for name, value := range m {
		if _, ok := value.([]interface{}); ok || reflect.TypeOf(value).String() == "[]interface {}" {
			for _, v := range value.([]interface{}) {
				err = h.appendValue(name, v, Base)
				if err != nil {
					return h, err
				}
			}
		} else if _, ok := value.(interface{}); ok {
			err = h.appendValue(name, value, Base)
			if err != nil {
				return h, err
			}
		} else {
			return h, fmt.Errorf(ErrFieldValueTypeNotValid, name)
		}
	}
	return h, err
}
