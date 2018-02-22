package sdk

import (
	"fmt"
)

type Scope string

var (
	ScopeOwn   Scope = "own"   // is read, add, edit and delete scopes combined
	ScopeWrite Scope = "write" // is add, edit and delete scopes combined

	ScopeRead   Scope = "read"
	ScopeAdd    Scope = "add"
	ScopeEdit   Scope = "edit"
	ScopeDelete Scope = "delete"
)

// todo: add ALS rules: read, write, ...
// todo: grouping (productName, productPrice, ...)
type Field struct {
	GroupName  string `json:"groupName"` // Deprecated
	Name       string `json:"name"`
	NoEdits    bool   `json:"noEdits"` // default true
	IsRequired bool   `json:"isRequired"`
	IsLat      bool   `json:"isLat"`
	IsLng      bool   `json:"isLng"`

	Type FieldType `json:"type"` // for special backend functions; file - saves multipart file and returns url
	// todo: search
	SearchType SearchType `json:"type"` // enables search if set

	Entity string `json:"-"`      // if set, value should be encoded entity key
	Lookup bool   `json:"lookup"` // if true, looks up entity value on output

	DefaultValue interface{}                   `json:"defaultValue"`
	ValueFunc    func() interface{}            `json:"-"`
	ContextFunc  func(ctx Context) interface{} `json:"-"`

	Multiple      bool                                                            `json:"multiple"`
	NoIndex       bool                                                            `json:"noIndex"`
	ValidateRgx   string                                                          `json:"validate"`
	Json          JsonOutput                                                      `json:"json"`
	TransformFunc func(ctx *ValueContext, value interface{}) (interface{}, error) `json:"-"`
	Validator     func(value interface{}) bool                                    `json:"-"`

	//GroupEntity GroupEntity `json:"groupEntity"`   // if defined, value stored as an separate entity; in field stored key
	Meta Meta `json:"meta"` // used for automatic admin html template creation

	SearchProps []interface{} `json:"-"`

	isSpecialField     bool   `json:"-"`
	datastoreFieldName string `json:"-"`
	fieldFunc []func(ctx *ValueContext, v interface{}) (interface{}, error) `json:"-"`
}

type FieldType string
type Meta map[string]interface{}

const (
	FileType  FieldType = "file"
	ImageType FieldType = "image"
	/*ImageType FieldType = "image"
	TextType FieldType = "text"
	HTMLTemplateType FieldType = "html"
	DateTimeType  FieldType = "datetime"
	BooleanType  FieldType = "bool"*/
)

type GroupEntity struct {
	Entity *Entity
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
