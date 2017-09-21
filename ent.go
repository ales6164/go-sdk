package sdk

import (
	"fmt"
)

type Ent struct {
	Name   string
	Fields map[string]*Field

	Rules Rules

	SearchIndex string // name of search index

	// listener
	/*OnWrite func(c *Conn) error
	OnRead  func(c *Conn) error*/
}

type Rules map[Scope]bool
type Scope string

var (
	UserRead  Scope = "user.read"
	UserEdit  Scope = "user.edit"
	UserAdd   Scope = "user.add"
	GuestRead Scope = "guest.read"
	GuestEdit Scope = "guest.edit"
	GuestAdd  Scope = "guest.add"
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