package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/log"
	"reflect"
	"regexp"
	"time"
	"github.com/google/uuid"
	"io/ioutil"
)

type Entity struct {
	Name    string
	Private bool  // protects entity with user field - only creator has access
	Cache   Cache // keeps values in memcache - good for categories, translations, ...

	Fields map[string]*Field
	fields []*Field

	hasFileFields bool
	parse map[string]func(ctx Context) (interface{}, error)

	preparedData map[*Field]func(ctx Context, f *Field) interface{}

	requiredFields []*Field

	indexes map[string]*DocumentDefinition

	// listeners
	OnAfterRead func(h *EntityDataHolder) error
}

type Cache struct {
	CacheOnWrite bool          // if true, caches data on write
	Expiration   time.Duration // value expiration date. Zero means no expiration
}

func NewEntity(name string, fields []*Field) *Entity {
	var e = new(Entity)

	if len(name) == 0 {
		panic(errors.New("entity name can't be empty"))
	}

	if name == "default" || name == "any" || name == "all" {
		panic(errors.New("entity name '" + name + "' is reserved and can't be used"))
	}

	e.Name = name
	e.fields = fields
	e.indexes = map[string]*DocumentDefinition{}

	e.preparedData = map[*Field]func(ctx Context, f *Field) interface{}{}

	e.Fields = map[string]*Field{}
	e.parse = map[string]func(ctx Context) (interface{}, error){}
	for _, field := range fields {
		if len(field.Name) == 0 {
			panic(errors.New("field name can't be empty"))
		}

		if field.Name == "_id" || field.Name == "id" {
			panic(errors.New("field name _id/id is reserved and can't be used"))
		}

		if field.Name[:1] == "_" {
			panic(errors.New("field name can't start with an underscore"))
		}

		e.addField(field)

		// todo
		if field.IsFile {
			e.parse[field.Name] = func(ctx Context) (interface{}, error) {
				return saveFile(ctx, field.Name)
			}
			e.hasFileFields = true
		}
	}

	// add special fields
	e.addField(&Field{
		Name:           "_createdAt",
		NoEdits:        true,
		NoIndex:        true,
		isSpecialField: true,
		ValueFunc: func() interface{} {
			return time.Now()
		},
	})
	e.addField(&Field{
		Name:           "_createdBy",
		NoEdits:        true,
		NoIndex:        true,
		isSpecialField: true,
		Entity:         userEntity,
		ContextFunc: func(ctx Context) interface{} {
			return ctx.User
		},
	})
	e.addField(&Field{
		Name:           "_updatedAt",
		NoIndex:        true,
		isSpecialField: true,
		ValueFunc: func() interface{} {
			return time.Now()
		},
	})
	e.addField(&Field{
		Name:           "_updatedBy",
		NoIndex:        true,
		isSpecialField: true,
		Entity:         userEntity,
		ContextFunc: func(ctx Context) interface{} {
			return ctx.User
		},
	})

	return e
}

func (e *Entity) addField(field *Field) {
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
		e.preparedData[field] = func(ctx Context, f *Field) interface{} {
			return f.DefaultValue
		}
	}

	if field.ValueFunc != nil {
		e.preparedData[field] = func(ctx Context, f *Field) interface{} {
			return f.ValueFunc()
		}
	}

	if field.ContextFunc != nil {
		e.preparedData[field] = func(ctx Context, f *Field) interface{} {
			return f.ContextFunc(ctx)
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

func (e *Entity) New(ctx Context) *EntityDataHolder {
	var dataHolder = &EntityDataHolder{
		Entity: e,
		data:   Data{},
		input:  map[string]interface{}{},
		isNew:  true,
	}

	// copy prepared values
	for field, fun := range e.preparedData {
		dataHolder.data[field] = fun(ctx, field)
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

	return c, key, err
}

func (e *Entity) NewIncompleteKey(c Context) (Context, *datastore.Key) {
	var key *datastore.Key

	key = datastore.NewIncompleteKey(c.Context, e.Name, nil)

	return c, key
}

// Gets appengine context and datastore key with optional namespace. It doesn't fail if request is not authenticated.
func (e *Entity) NewKey(c Context, nameId interface{}) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	if nameId == nil {
		return c, key, ErrKeyNameIdNil
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
	var h = e.New(c)

	// todo: fix this
	c.r.FormValue("a")

	// e.parse only parses form values for now
	/*for fieldName, fun := range e.parse {
		val, err := fun(c)
		if err != nil {
			return h, err
		}
		err = h.appendValue(fieldName, val, Low)
		if err != nil {
			return h, err
		}
	}*/

	if e.hasFileFields {
		c.r.ParseMultipartForm(32 << 20)
		m := c.r.MultipartForm
		for name, v := range m.File {
			if _, ok := e.parse[name]; ok {
				for _, f := range v {
					file, err := f.Open()
					if err != nil {
						return h, err
					}

					fileKeyName := uuid.New().String()
					bytes, err := ioutil.ReadAll(file)
					file.Close()
					if err != nil {
						return h, err
					}
					url, err := writeFile(c.Context, fileKeyName, f.Filename, bytes)
					if err != nil {
						return h, err
					}

					/*log.Infof(c.Context, "Appending file url '%s' value: %s", name, url)*/

					err = h.appendValue(name, "https://storage.googleapis.com/"+bucketName+"/"+url, Low)
					if err != nil {
						return h, err
					}
				}
			}
		}
	}

	err := c.r.ParseForm()
	if len(c.r.Form) != 0 {
		for name, values := range c.r.Form {
			// remove '[]' from fieldName if it's an array
			if len(name) > 2 && name[len(name)-2:] == "[]" {
				name = name[:len(name)-2]
			}

			for _, v := range values {
				/*log.Infof(c.Context, "Appending '%s' value: %v", name, v)*/

				err = h.appendValue(name, v, Low)
				if err != nil {
					return h, err
				}
			}
		}
	} else if len(c.r.PostForm) != 0 {
		for name, values := range c.r.PostForm {
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
	var err error

	c = c.WithBody()

	if len(c.body.body) == 0 {
		return e.New(c), nil
	}

	var t map[string]interface{}
	err = json.Unmarshal(c.body.body, &t)
	if err != nil {
		return e.New(c), err
	}

	return e.FromMap(c, t)
}

func (e *Entity) FromMap(c Context, m map[string]interface{}) (*EntityDataHolder, error) {
	var h = e.New(c)
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
