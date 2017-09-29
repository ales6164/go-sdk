package sdk

import (
	"google.golang.org/appengine/datastore"
	"regexp"
	"errors"
	"reflect"
)


type DataObject struct {
	DataMap map[string]interface{}
	Output  datastore.PropertyList
}

var (
	ErrKeyNameIdNil         = errors.New("key nameId is nil")
	ErrKeyNameIdInvalidType = errors.New("key nameId invalid type (only string/int64)")
)

func (e *PreparedEntity) DecodeKey(c Context, encodedKey string) (Context, *datastore.Key, error) {
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

func (e *PreparedEntity) NewIncompleteKey(c Context, withNamespace bool) (Context, *datastore.Key) {
	var key *datastore.Key

	if withNamespace {
		c.WithNamespace()
	}

	key = datastore.NewIncompleteKey(c.Context, e.Entity.Name, nil)

	return c, key
}

// Gets appengine context and datastore key with optional namespace. It doesn't fail if request is not authenticated.
func (e *PreparedEntity) NewKey(c Context, nameId interface{}, withNamespace bool) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	if nameId == nil {
		return c, key, ErrKeyNameIdNil
	}

	if withNamespace {
		c.WithNamespace()
	}

	if stringId, ok := nameId.(string); ok {
		key = datastore.NewKey(c.Context, e.Entity.Name, stringId, 0, nil)
	} else if intId, ok := nameId.(int64); ok {
		key = datastore.NewKey(c.Context, e.Entity.Name, "", intId, nil)
	} else {
		return c, key, ErrKeyNameIdInvalidType
	}

	return c, key, err
}

var (
	ErrInvalidType        = errors.New("data: invalid type")
	ErrMultipleProperties = errors.New("data: multiple properties")
)

func (e *PreparedEntity) FromMap(c Context, dataMap map[string]interface{}) (*DataObject, error) {
	var err error
	var dataObject = new(DataObject)
	dataObject.DataMap = map[string]interface{}{}

	// copy values
	copy(dataObject.Output, e.Output)

	// copy values
	for key, value := range e.Input {
		dataObject.DataMap[key] = value
	}

	// append ready fields
	e.appendReadyFields(dataObject)

	for name, values := range dataMap {

		// remove '[]' from fieldName if it's an array
		if len(name) > 2 && name[len(name)-2:] == "[]" {
			name = name[:len(name)-2]
		}

		if _, ok := values.([]interface{}); ok || reflect.TypeOf(values).String() == "[]interface {}" {
			for _, v := range values.([]interface{}) {
				err = e.rangeOverData(name, v, dataObject)
				if err != nil {
					return dataObject, err
				}
			}
		} else if valuesInt, ok := values.(interface{}); ok {
			err = e.rangeOverData(name, valuesInt, dataObject)
			if err != nil {
				return dataObject, err
			}
		} else {
			return dataObject, ErrInvalidType
		}
	}

	for _, fieldName := range e.RequiredFields {
		if _, ok := dataObject.DataMap[fieldName]; !ok {
			return dataObject, FieldRequired.Params(fieldName)
		}
	}

	return dataObject, err

}

func (e *PreparedEntity) FromForm(c Context) (*DataObject, error) {
	var err error
	var dataObject = new(DataObject)
	dataObject.DataMap = map[string]interface{}{}

	copy(dataObject.Output, e.Output)

	// copy values
	for key, value := range e.Input {
		dataObject.DataMap[key] = value
	}

	// append ready fields
	e.appendReadyFields(dataObject)

	// todo: fix this
	c.r.FormValue("a")

	if err = c.r.ParseForm(); err != nil {
		return dataObject, err
	}

	for name, values := range c.r.Form {

		// remove '[]' from fieldName if it's an array
		if len(name) > 2 && name[len(name)-2:] == "[]" {
			name = name[:len(name)-2]
		}

		for _, v := range values {
			err = e.rangeOverData(name, v, dataObject)
			if err != nil {
				return dataObject, err
			}
		}

	}

	for _, fieldName := range e.RequiredFields {
		if _, ok := dataObject.DataMap[fieldName]; !ok {
			return dataObject, FieldRequired.Params(fieldName)
		}
	}

	return dataObject, err

	/*// set search query
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
	}*/
}

func (e *PreparedEntity) appendReadyFields(dataObj *DataObject) {
	for field, fun := range e.Ready {
		/*if field.Multiple {
			dataObj.Input[field.Name] = []interface{}{}

			for _, funFun := range fun.(func() interface{}) {
				var value = funFun()
				dataObj.Input[field.Name] = append(dataObj.Input[field.Name].([]interface{}), value)
				dataObj.Output = append(dataObj.Output, datastore.Property{
					Name:     field.Name,
					Value:    value,
					NoIndex:  field.NoIndex,
					Multiple: field.Multiple,
				})
			}
		} else {*/
		var value = fun.(func() interface{})()

		if field.Json != NoJsonOutput {
			dataObj.DataMap[string(field.Json)] = value
		}

		dataObj.Output = append(dataObj.Output, datastore.Property{
			Name:     field.Name,
			Value:    value,
			NoIndex:  field.NoIndex,
			Multiple: field.Multiple,
		})
		/*}*/
	}
}

func (e *PreparedEntity) rangeOverData(name string, value interface{}, dataObj *DataObject) error {
	var field *Field
	var err error

	if field, value, err = validateAndTransformFieldValue(e.Entity, name, value); err != nil {
		return err
	}

	if field != nil {

		if field.Multiple {
			if _, ok := dataObj.DataMap[name]; !ok {
				dataObj.DataMap[name] = []interface{}{}
			}
			dataObj.DataMap[name] = append(dataObj.DataMap[name].([]interface{}), value)
		} else {
			dataObj.DataMap[name] = value
		}

		dataObj.Output = append(dataObj.Output, datastore.Property{
			Name:     field.Name,
			Value:    value,
			NoIndex:  field.NoIndex,
			Multiple: field.Multiple,
		})
	} else {
		// if we get multiple values with the same name (from undefined field) we create an array
		if currData, ok := dataObj.DataMap[name]; ok {

			// check if already is an array
			if _, ok := currData.([]interface{}); ok {
				dataObj.DataMap[name] = append(dataObj.DataMap[name].([]interface{}), value)
			} else {
				dataObj.DataMap[name] = []interface{}{currData}
			}
		}
	}
	return nil
}

func validateAndTransformFieldValue(entity *Ent, name string, value interface{}) (*Field, interface{}, error) {
	if value != nil {

		// get named field
		if field, ok := entity.Fields[name]; ok {

			// validate rgx
			if len(field.ValidateRgx) > 0 {
				if matched, err := regexp.Match(field.ValidateRgx, []byte(value.(string))); !matched || err != nil {
					return field, value, err
				}
			}

			// validate func
			if field.Validator != nil {
				if ok := field.Validator(value); !ok {
					return field, value, FieldValidationError.Params(field.Name)
				}
			}

			// transform
			if field.TransformFunc != nil {
				var err error
				if value, err = field.TransformFunc(value); err != nil {
					return field, value, err
				}
			}

			return field, value, nil
		}
	}
	return nil, value, nil
}

func (d *DataObject) Add(e *PreparedEntity, name string, value interface{}) {
	if field, ok := e.Entity.Fields[name]; ok {
		if field.Multiple {
			if _, ok := d.DataMap[name]; !ok {
				d.DataMap[name] = []interface{}{}
			}
			d.DataMap[name] = append(d.DataMap[name].([]interface{}), value)
		} else {
			d.DataMap[name] = value
		}

		d.Output = append(d.Output, datastore.Property{
			Name:     field.Name,
			Value:    value,
			NoIndex:  field.NoIndex,
			Multiple: field.Multiple,
		})
	} else {
		if currData, ok := d.DataMap[name]; ok {

			// check if already is an array
			if _, ok := currData.([]interface{}); ok {
				d.DataMap[name] = append(d.DataMap[name].([]interface{}), value)
			} else {
				d.DataMap[name] = []interface{}{currData}
			}
		}
	}
}
