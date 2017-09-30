package sdk

import (
	"fmt"
)

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
	GroupName  string `json:"groupName"`
	Name       string `json:"name"`
	NoEdits    bool   `json:"noEdits"` // default true
	IsRequired bool   `json:"isRequired"`

	DefaultValue interface{}        `json:"defaultValue"`
	ValueFunc    func() interface{} `json:"valueFunc"`

	Multiple      bool                                                            `json:"multiple"`
	NoIndex       bool                                                            `json:"noIndex"`
	ValidateRgx   string                                                          `json:"validate"`
	Json          JsonOutput                                                      `json:"json"`
	TransformFunc func(ctx *ValueContext, value interface{}) (interface{}, error) `json:"-"`
	Validator     func(value interface{}) bool                                    `json:"-"`

	SearchProps []interface{} `json:"-"`

	datastoreFieldName string
	fieldFunc []func(ctx *ValueContext, v interface{}) (interface{}, error)
}

type ValueContext struct {
	Trust ValueTrust
	Field *Field
}

type ValueTrust string

const (
	Low  ValueTrust = "low"
	Base ValueTrust = "base"
	High ValueTrust = "high"
)

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
	EntityAlreadyExists = NewError("connection: entity %s already exists")
	Unauthorized        = NewError("connection: authorization error: %s")
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
