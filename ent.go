package sdk

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"github.com/asaskevich/govalidator"
	"google.golang.org/appengine/search"
	"net/http"
	gcontext"github.com/gorilla/context"
	"github.com/dgrijalva/jwt-go"
	"fmt"
	"bytes"
	"encoding/json"
)

type Ent struct {
	Name   string
	Fields map[string]*Field

	Key       *Key
	ParentKey *Key

	Rules Rules

	SearchIndex string // name of search index

	Data []datastore.Property // only Load/Save should write

	// listener
	OnWrite func(c *Conn) error
	OnRead  func(c *Conn) error

	EncodedKey string
}

type Rules map[Scope]bool
type Scope string

var (
	UserWrite  Scope = "user.write"
	UserRead   Scope = "user.read"
	UserEdit   Scope = "user.edit"
	UserAdd    Scope = "user.add"
	GuestWrite Scope = "guest.write"
	GuestRead  Scope = "guest.read"
	GuestEdit  Scope = "guest.edit"
	GuestAdd   Scope = "guest.add"
)

var UserKind string = "_User_"

type Conn struct {
	Context context.Context
	Entity  *Ent
	err     error
	user    string
	/*	id        interface{}*/
	key       *Key
	parentKey *Key
	r         *http.Request
	InputData map[string][]interface{}
	QueryData map[string][]interface{}

	Data      map[string][]interface{}
	MultiData []map[string]interface{}
}

type Key struct {
	NamespaceType
	Kind      string
	FromField string
	FromToken bool
	StringID  string
	IntID     int64
}

// todo: add ALS rules: read, write, ...
// todo: grouping (productName, productPrice, ...)
type Field struct {
	GroupName       string                                       `json:"groupName"`
	Name            string                                       `json:"name"`
	IsRequired      bool                                         `json:"isRequired"`
	DefaultValue    interface{}                                  `json:"defaultValue"`
	WithValueFunc   func() interface{}                           `json:"withValueFunc"`
	WithStaticValue interface{}                                  `json:"withstaticValue"`
	Multiple        bool                                         `json:"multiple"`
	NoIndex         bool                                         `json:"noIndex"`
	ValidateRgx     string                                       `json:"validate"`
	Json            JsonOutput                                   `json:"json"`
	TransformFunc   func(value interface{}) (interface{}, error) `json:"-"`
	Validator       func(value interface{}) bool                 `json:"-"`

	SearchProps []interface{} `json:"-"`
}

type JsonOutput string

const (
	NoJsonOutput JsonOutput = "-"
)

type SearchField struct {
	Name          string
	Derived       bool
	Language      string
	TransformFunc func(value interface{}) (interface{}, error) `json:"-"`
}

type SearchFacet struct {
	Name          string
	TransformFunc func(value interface{}) (interface{}, error) `json:"-"`
}

var (
	EntityAlreadyExists      = NewError("connection: entity %s already exists")
	FieldUndefined           = NewError("connection: undefined field %s")
	FieldRequired            = NewError("connection: field %s required")
	FieldValidationError     = NewError("connection: field %s is not valid")
	RequestMissingIdField    = NewError("connection: entity %s request is missing id field")
	FieldHasMoreThanOneValue = NewError("entity: field %s has more than one value")
	IdFieldValueTypeError    = NewError("entity: id field %s value has to be of type string or int64")
	NoIdFieldValue           = NewError("entity: id field %s is empty")
	InvalidNamespaceType     = NewError("entity: invalid namespace type %v")
	ValueFuncOfInvalidType   = NewError("entity: value func %s of invalid type")
	Unauthorized             = NewError("connection: authorization error: %s")
	GuestAccessRequest       = NewError("request: guest access")
	ErrDecodingKey           = NewError("connection: error decoding key")
	ErrFetching              = NewError("fetch: error fetching next %v")
)

type NamespaceType string

const (
	NoNamespace   NamespaceType = ""
	UserNamespace NamespaceType = "userNamespace"
)

type Error struct {
	s string
	p []interface{}
}

func (e *Error) Error() string {
	return fmt.Sprintf(e.s, e.p...)
}

func (e *Error) Params(values ...interface{}) *Error {
	e.p = values
	return e
}

func NewError(text string) *Error {
	return &Error{s: text}
}

func NewEntity(name string, fields []*Field) *Ent {
	var ent = &Ent{}

	var fs = map[string]*Field{}
	for _, field := range fields {
		fs[field.Name] = field
	}

	ent.Name = name
	ent.Fields = fs

	return ent
}

func (e *Ent) NewConnection(ctx context.Context) *Conn {
	return &Conn{Context: ctx, Entity: e}
}

func (e *Ent) WithData(data map[string][]interface{}) ([]datastore.Property, error) {
	var assembled []datastore.Property
	for name, values := range data {
		var field *Field
		var ok bool
		if field, ok = e.Fields[name]; !ok {
			continue
		}

		if !field.Multiple && len(values) > 1 {
			return assembled, FieldHasMoreThanOneValue.Params(field.Name)
		}

		for _, val := range values {

			if len(field.GroupName) > 0 && val == nil {
				if field.IsRequired {
					return assembled, FieldRequired.Params(field.Name)
				}
				if field.DefaultValue != nil {
					val = field.DefaultValue
				}
			}

			assembled = append(assembled, datastore.Property{
				Name:     field.Name,
				Value:    val,
				Multiple: field.Multiple,
				NoIndex:  field.NoIndex,
			})
		}
	}
	return assembled, nil
}

func (c *Conn) WithRequest(r *http.Request) (*Conn, error) {
	c.r = r

	// set input data
	c.InputData = map[string][]interface{}{}
	c.err = r.ParseForm()
	if c.err != nil {
		return c, c.err
	}
	for name, values := range r.Form {
		for _, val := range values {
			c.InputData[name] = append(c.InputData[name], val)
		}
	}

	// set request token
	requestToken := gcontext.Get(r, "user")
	if requestToken != nil {
		var ok bool
		c.user, ok = resolveToken(requestToken)
		if !ok {
			c.user = ""
			c.err = Unauthorized.Params("invalid token")
			return c, c.err
		} else {
			c.parentKey = &Key{Kind: UserKind, StringID: c.user, IntID: 0}
		}
	}

	// set search query
	c.QueryData = map[string][]interface{}{}
	for param, val := range r.URL.Query() {
		c.QueryData[param] = append(c.QueryData[param], val)
	}

	// set input data from body
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	defer r.Body.Close()
	if buf.Len() > 0 {
		c.err = json.Unmarshal(buf.Bytes(), &c.InputData)
		return c, c.err
	}

	return c, c.err
}

func (e *Ent) GetData() map[string]interface{} {
	var data = map[string]interface{}{}

	for _, d := range e.Data {
		if _, ok := data[d.Name]; !ok && d.Multiple {
			data[d.Name] = []interface{}{}
		}

		if d.Multiple {
			data[d.Name] = append(data[d.Name].([]interface{}), d.Value)
		} else {
			data[d.Name] = d.Value
		}
	}

	return data
}

func (c *Conn) Key(nameID string, intID int64) *Conn {
	c.key = &Key{Kind: c.Entity.Name, StringID: nameID, IntID: intID}
	return c
}

/*func (c *Conn) Parent(kind string, nameID string, intID int64) *Conn {
	if c.parentKey != nil && c.parentKey.Kind == UserKind {
		c.err = RedeclaringUnathorizedParent
		return c
	}
	c.parentKey = &Key{kind, nameID, intID}
	return c
}*/

func (c *Conn) Get() (string, map[string]interface{}, error) {
	var nilId string
	var data map[string]interface{}
	var err error

	if c.err != nil {
		return nilId, data, c.err
	}

	if c.Entity.Rules[UserRead] {
		if c.parentKey != nil && c.parentKey.Kind != UserKind {
			return nilId, data, Unauthorized.Params("scope")
		}
	} else if c.Entity.Rules[GuestRead] {
		if c.parentKey != nil && c.parentKey.Kind == UserKind {
			return nilId, data, Unauthorized.Params("scope")
		}
		c.key = nil
	} else {
		return nilId, data, Unauthorized.Params("scope")
	}

	key, err := c.getKey(c.key, c.parentKey)
	if err != nil {
		return nilId, data, err
	}

	err = datastore.Get(c.Context, key, c.Entity)
	if err != nil {
		return nilId, c.Entity.GetData(), err
	}

	if c.Entity.OnRead != nil {
		err = c.Entity.OnRead(c)
		if err != nil {
			return nilId, c.Entity.GetData(), err
		}
	}

	return key.Encode(), c.Entity.GetData(), nil
}

func (c *Conn) Query() ([]map[string]interface{}, error) {
	var err error

	if c.err != nil {
		return c.MultiData, c.err
	}

	if c.Entity.Rules[UserRead] {
		if c.parentKey != nil && c.parentKey.Kind != UserKind {
			return c.MultiData, Unauthorized.Params("scope")
		}
	} else if c.Entity.Rules[GuestRead] {
		if c.parentKey != nil && c.parentKey.Kind == UserKind {
			return c.MultiData, Unauthorized.Params("scope")
		}
		c.key = nil
	} else {
		return c.MultiData, Unauthorized.Params("scope")
	}

	q := datastore.NewQuery(c.Entity.Name)
	for param, valArr := range c.QueryData {
		for _, val := range valArr {
			switch param {
			case "limit":
				q = q.Limit(val.(int))
				break
			case "order":
				q = q.Order(val.(string))
				break
			case "offset":
				q = q.Offset(val.(int))
				break
			}
		}
	}

	t := q.Run(c.Context)
	for {
		k, err := t.Next(c.Entity)
		if err == datastore.Done {
			break // No further entities match the query.
		}
		if err != nil {
			return c.MultiData, err
		}
		// Do something with Person p and Key k
		c.Entity.EncodedKey = k.Encode()

		c.MultiData = append(c.MultiData, c.Entity.GetData())
	}

	if c.Entity.OnRead != nil {
		err = c.Entity.OnRead(c)
		if err != nil {
			return c.MultiData, err
		}
	}

	return c.MultiData, nil
}

func (c *Conn) Put(unique bool) (string, map[string]interface{}, error) {
	var nilId string
	var data map[string]interface{}
	/*var searchFields []search.Field
	var searchFacets []search.Facet*/
	var err error

	if c.err != nil {
		return nilId, data, c.err
	}

	if c.Entity.Rules[UserWrite] || c.Entity.Rules[UserAdd] {
		if c.parentKey != nil && c.parentKey.Kind != UserKind {
			return nilId, data, Unauthorized.Params("scope")
		}
	} else if c.Entity.Rules[GuestWrite] || c.Entity.Rules[GuestAdd] {
		if c.parentKey != nil && c.parentKey.Kind == UserKind {
			return nilId, data, Unauthorized.Params("scope")
		}
		c.key = nil
	} else {
		return nilId, data, Unauthorized.Params("scope")
	}

	var props []datastore.Property
	props, err = c.Entity.WithData(c.InputData)
	if err != nil {
		return nilId, data, err
	}

	c.Entity.Data, _, _, err = checkWithFields(c.Entity.Fields, props)
	if err != nil {
		return nilId, data, err
	}

	key, err := c.getKey(c.key, c.parentKey)
	if err != nil {
		return nilId, data, err
	}

	if unique && !key.Incomplete() {
		var alreadyExists bool
		err = datastore.RunInTransaction(c.Context, func(tc context.Context) error {
			var tempEnt Ent
			err := datastore.Get(tc, key, &tempEnt)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					if c.Entity.OnWrite != nil {
						err = c.Entity.OnWrite(c)
						if err != nil {
							return err
						}
					}
					key, err = datastore.Put(tc, key, c.Entity)
					return err
				}
				return err
			} else {
				alreadyExists = true
			}
			return nil
		}, nil)

		if alreadyExists {
			return nilId, data, EntityAlreadyExists.Params(c.Entity.Name)
		}
	} else {
		key, err = datastore.Put(c.Context, key, c.Entity)
	}

	// todo: search put
	/*if len(searchFacets) > 0 || len(searchFields) > 0 {
		index, err := search.Open(c.ent.SearchIndex)
		if err != nil {
			return data, err
		}
		index.Put(c.ctx, key.Encode(), )
	}*/

	return key.Encode(), c.Entity.GetData(), err
}

func (c *Conn) getKey(k *Key, p *Key) (*datastore.Key, error) {
	var parentKey *datastore.Key
	var key *datastore.Key

	if c.parentKey != nil && (c.parentKey.IntID > 0 || len(c.parentKey.StringID) > 0) {
		parentKey = datastore.NewKey(c.Context, c.parentKey.Kind, c.parentKey.StringID, c.parentKey.IntID, nil)
	}
	if c.key != nil && (c.key.IntID > 0 || len(c.key.StringID) > 0) {
		key = datastore.NewKey(c.Context, c.key.Kind, c.key.StringID, c.key.IntID, parentKey)
	} else {
		key = datastore.NewIncompleteKey(c.Context, c.Entity.Name, parentKey)
	}

	return key, nil
}

func checkWithFields(fields map[string]*Field, props []datastore.Property) ([]datastore.Property, []search.Field, []search.Facet, error) {
	var data []datastore.Property
	var sFields []search.Field
	var sFacets []search.Facet
	var err error

	var doneFields = map[string]bool{}

	for _, prop := range props {

		var field *Field

		// check if value field is defined
		var ok bool
		if field, ok = fields[prop.Name]; !ok {
			return data, sFields, sFacets, FieldUndefined.Params(field.Name)
		}

		if field.WithStaticValue != nil {
			continue
		}

		// check if value is nil and add default value
		strVal, isString := prop.Value.(string)
		if prop.Value == nil || (isString && len(strVal) == 0) {
			if field.IsRequired {
				return data, sFields, sFacets, FieldRequired.Params(field.Name)
			}
			if field.DefaultValue == nil {
				continue
			}
			prop.Value = field.DefaultValue
		}

		prop, err = validateValueWithField(field, prop)
		if err != nil {
			return data, sFields, sFacets, err
		}

		prepFields, prepFacets, err := prepareSearchDoc(field, prop)
		if err != nil {
			return data, sFields, sFacets, err
		}
		sFields = append(sFields, prepFields...)
		sFacets = append(sFacets, prepFacets...)

		data = append(data, prop)

		doneFields[field.Name] = true
	}

	// check if all fields are used
	for _, field := range fields {
		if doneFields[field.Name] {
			continue
		}

		if field.IsRequired {
			return data, sFields, sFacets, FieldRequired.Params(field.Name)
		}

		var prop = datastore.Property{
			Name: field.Name,
		}

		if field.WithStaticValue != nil {
			prop.Value = field.WithStaticValue
		} else if field.DefaultValue != nil {
			prop.Value = field.DefaultValue
		}

		prop, err = validateValueWithField(field, prop)
		if err != nil {
			return data, sFields, sFacets, err
		}

		prepFields, prepFacets, err := prepareSearchDoc(field, prop)
		if err != nil {
			return data, sFields, sFacets, err
		}
		sFields = append(sFields, prepFields...)
		sFacets = append(sFacets, prepFacets...)

		data = append(data, prop)
	}

	return data, sFields, sFacets, err
}

func validateValueWithField(field *Field, prop datastore.Property) (datastore.Property, error) {
	if prop.Value != nil {

		// validate rgx
		if len(field.ValidateRgx) > 0 {
			if matches := govalidator.Matches(prop.Value.(string), field.ValidateRgx); !matches {
				return prop, FieldValidationError.Params(field.Name)
			}
		}

		// validate func
		if field.Validator != nil {
			if ok := field.Validator(prop.Value); !ok {
				return prop, FieldValidationError.Params(field.Name)
			}
		}

		// transform
		if field.TransformFunc != nil {
			transformed, err := field.TransformFunc(prop.Value)
			if err != nil {
				return prop, err
			}

			prop.Value = transformed
		}
	}

	prop.Multiple = field.Multiple
	prop.NoIndex = field.NoIndex

	return prop, nil
}

func prepareSearchDoc(field *Field, prop datastore.Property) ([]search.Field, []search.Facet, error) {
	var fields []search.Field
	var facets []search.Facet
	var err error

	for _, sf := range field.SearchProps {
		var val = prop.Value

		if facet, ok := sf.(SearchFacet); ok {
			if facet.TransformFunc != nil && val != nil {
				val, err = facet.TransformFunc(val)
				if err != nil {
					return fields, facets, err
				}
			}

			facets = append(facets, search.Facet{
				Name:  facet.Name,
				Value: val,
			})
		} else if field, ok := sf.(SearchField); ok {
			if field.TransformFunc != nil && val != nil {
				val, err = field.TransformFunc(val)
				if err != nil {
					return fields, facets, err
				}
			}

			fields = append(fields, search.Field{
				Name:     field.Name,
				Value:    val,
				Derived:  field.Derived,
				Language: field.Language,
			})
		}
	}

	return fields, facets, nil
}

func resolveToken(userToken interface{}) (string, bool) {
	var email string
	var claims jwt.MapClaims
	var ok bool
	token := userToken.(*jwt.Token)

	if claims, ok = token.Claims.(jwt.MapClaims); ok && token.Valid {
		if email, ok = claims["sub"].(string); !ok {
			return email, false
		}
		return email, true
	}
	return email, false
}

func (e *Ent) Load(ps []datastore.Property) error {
	e.Data = ps
	return nil
}

func (e *Ent) Save() ([]datastore.Property, error) {
	return e.Data, nil
}
