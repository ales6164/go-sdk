package sdk

import (
	"fmt"
	"google.golang.org/appengine/datastore"
)

type Ent struct {
	Name   string
	Fields map[string]*Field

	// listener
	/*OnWrite func(c *Conn) error
	OnRead  func(c *Conn) error*/
}

type Scope string

var (
	ScopeGet    Scope = "get"
	ScopeEdit   Scope = "edit"
	ScopePut    Scope = "put"
	ScopeAdd    Scope = "add"
	ScopeDelete Scope = "delete"
)

// todo: add ALS rules: read, write, ...
// todo: grouping (productName, productPrice, ...)
type Field struct {
	GroupName       string                                       `json:"groupName"`
	Name            string                                       `json:"name"`
	IsRequired      bool                                         `json:"isRequired"`
	DefaultValue    interface{}                                  `json:"defaultValue"`
	WithValueFunc   func() interface{}                           `json:"withValueFunc"`
	WithStaticValue interface{}                                  `json:"withStaticValue"`
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

func NewEntity(name string, fields []*Field) *Ent {
	var ent = new(Ent)
	ent.Name = name

	ent.Fields = map[string]*Field{}
	for _, field := range fields {
		ent.Fields[field.Name] = field
	}

	return ent
}

func (e *Ent) Prepare() *PreparedEntity {
	var prepared = new(PreparedEntity)
	prepared.Entity = e
	prepared.Ready = map[*Field]interface{}{}
	prepared.Input = map[string]interface{}{}

	for _, field := range e.Fields {

		if len(field.Json) == 0 {
			field.Json = JsonOutput(field.Name)
		}

		if field.IsRequired {
			prepared.RequiredFields = append(prepared.RequiredFields, field.Name)
		}

		if field.WithStaticValue != nil {
			if field.Multiple {
				if prepared.Input[field.Name] == nil {
					prepared.Input[field.Name] = []interface{}{}
				}
				prepared.Input[field.Name] = append(prepared.Input[field.Name].([]interface{}), field.WithStaticValue)
			} else {
				prepared.Input[field.Name] = field.WithStaticValue
			}

			prepared.Output = append(prepared.Output, datastore.Property{
				Name:     field.Name,
				Value:    field.WithStaticValue,
				NoIndex:  field.NoIndex,
				Multiple: field.Multiple,
			})
		}

		if field.WithValueFunc != nil {
			if field.Multiple {
				if prepared.Ready[field] == nil {
					prepared.Ready[field] = []func() interface{}{}
				}
				prepared.Ready[field] = append(prepared.Ready[field].([]func() interface{}), field.WithValueFunc)
			} else {
				prepared.Ready[field] = field.WithValueFunc
			}
		}
	}

	return prepared
}

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
