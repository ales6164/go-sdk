package sdk

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/appengine/search"
	"strconv"
)

func FuncToFloatTransform(c *ValueContext, value interface{}) (interface{}, error) {
	if _, ok := value.(float64); ok {
		return value, nil
	}
	return strconv.ParseFloat(value.(string), 64)
}

func FuncToIntTransform(c *ValueContext, value interface{}) (interface{}, error) {
	if _, ok := value.(float64); ok {
		return value, nil
	}
	return strconv.ParseInt(value.(string), 10, 64)
}

func FuncToAtomTransform(c *ValueContext, value interface{}) (interface{}, error) {
	if _, ok := value.(string); !ok {
		return value, errors.New("atom-transform: value not of type string")
	}
	return search.Atom(value.(string)), nil
}

func FuncHashTransform(c *ValueContext, value interface{}) (interface{}, error) {
	return crypt([]byte(value.(string)))
}

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, 13)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
