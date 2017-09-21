package sdk

import (
	"google.golang.org/appengine/datastore"
	"regexp"
	"google.golang.org/appengine"
	"errors"
)

type PreparedEntity struct {
	Input          map[string]interface{}
	Ready          map[*Field]interface{}
	Output         datastore.PropertyList
	RequiredFields []string
	Entity         *Ent
}

// Fills static data and assembles keys
func PrepareEntity(ent *Ent) *PreparedEntity {
	var prepared = new(PreparedEntity)
	prepared.Entity = ent
	prepared.Ready = map[*Field]interface{}{}
	prepared.Input = map[string]interface{}{}

	for fieldName, field := range ent.Fields {

		if field.IsRequired {
			prepared.RequiredFields = append(prepared.RequiredFields, fieldName)
		}

		if len(field.Json) == 0 && field.Json != NoJsonOutput {
			field.Json = JsonOutput(fieldName)
		}

		if field.WithStaticValue != nil {
			if field.Multiple {
				if prepared.Input[fieldName] == nil {
					prepared.Input[fieldName] = []interface{}{}
				}
				prepared.Input[fieldName] = append(prepared.Input[fieldName].([]interface{}), field.WithStaticValue)
			} else {
				prepared.Input[fieldName] = field.WithStaticValue
			}

			prepared.Output = append(prepared.Output, datastore.Property{
				Name:     fieldName,
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

type DataObject struct {
	Input  map[string]interface{}
	Output datastore.PropertyList
}

func (e *PreparedEntity) appendReadyFields(dataObj *DataObject) {
	for field, fun := range e.Ready {
		if field.Multiple {
			dataObj.Input[field.Name] = []interface{}{}

			for _, funFun := range fun.([]func() interface{}) {
				var value = funFun()
				dataObj.Input[field.Name] = append(dataObj.Input[field.Name].([]interface{}), value)
				dataObj.Output = append(dataObj.Output, datastore.Property{
					Name:     field.Name,
					Value:    value,
					NoIndex:  field.NoIndex,
					Multiple: field.Multiple,
				})
			}
		} else {
			var value = fun.(func() interface{})()

			if len(field.Json) > 0 {
				if field.Json != NoJsonOutput {
					dataObj.Input[string(field.Json)] = value
				}
			} else {
				dataObj.Input[field.Name] = value
			}

			dataObj.Output = append(dataObj.Output, datastore.Property{
				Name:     field.Name,
				Value:    value,
				NoIndex:  field.NoIndex,
				Multiple: field.Multiple,
			})
		}
	}
}

func (e *PreparedEntity) rangeOverData(name string, value interface{}, dataObj *DataObject) error {
	var field *Field
	var err error

	if field, value, err = validateAndTransformFieldValue(e.Entity, name, value); err != nil {
		return err
	}

	if field != nil {
		var noJsonOutput bool
		if len(field.Json) > 0 {
			if field.Json == NoJsonOutput {
				noJsonOutput = true
			} /*else {
				name = string(field.Json)
			}*/
		}

		if !noJsonOutput {
			if field.Multiple {
				if _, ok := dataObj.Input[name]; !ok {
					dataObj.Input[name] = []interface{}{}
				}
				dataObj.Input[name] = append(dataObj.Input[name].([]interface{}), value)
			} else {
				dataObj.Input[name] = value
			}
		}

		dataObj.Output = append(dataObj.Output, datastore.Property{
			Name:     field.Name,
			Value:    value,
			NoIndex:  field.NoIndex,
			Multiple: field.Multiple,
		})
	} else {
		// if we get multiple values with the same name (from undefined field) we create an array
		if currData, ok := dataObj.Input[name]; ok {

			// check if already is an array
			if _, ok := currData.([]interface{}); ok {
				dataObj.Input[name] = append(dataObj.Input[name].([]interface{}), value)
			} else {
				dataObj.Input[name] = []interface{}{currData}
			}
		}
	}
	return nil
}

func (e *PreparedEntity) FromMap(c Context, dataMap map[string]interface{}, checkIfRequired bool) (*DataObject, error) {
	var err error
	var dataObject = &DataObject{
		Input:  map[string]interface{}{},
		Output: e.Output,
	}

	// copy values
	for key, value := range e.Input {
		dataObject.Input[key] = value
	}

	// append ready fields
	e.appendReadyFields(dataObject)

	for name, values := range dataMap {

		// remove '[]' from fieldName if it's an array
		if len(name) > 2 && name[len(name)-2:] == "[]" {
			name = name[:len(name)-2]
		}

		if valuesArr, ok := values.([]interface{}); ok {
			for _, v := range valuesArr {
				err = e.rangeOverData(name, v, dataObject)
				if err != nil {
					return dataObject, err
				}
			}
		} else {
			err = e.rangeOverData(name, values, dataObject)
			if err != nil {
				return dataObject, err
			}
		}
	}

	return dataObject, err

}

func (e *PreparedEntity) FromForm(c Context) (*DataObject, error) {
	var err error
	var dataObject = &DataObject{
		Input:  map[string]interface{}{},
		Output: e.Output,
	}

	// copy values
	for key, value := range e.Input {
		dataObject.Input[key] = value
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
		if _, ok := dataObject.Input[fieldName]; !ok {
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

var (
	ErrKeyNameIdNil         = errors.New("key nameId is nil")
	ErrKeyNameIdInvalidType = errors.New("key nameId invalid type (only string/int64)")
)

func (e *PreparedEntity) NewIncompleteKey(c Context, withNamespace bool) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	c.ctx = appengine.NewContext(c.r)

	if withNamespace {
		if c.isAuthenticated && len(c.namespace) > 0 {
			c.ctx, err = appengine.Namespace(c.ctx, c.namespace)
			if err != nil {
				return c, key, err
			}
		} else {
			err = ErrNotAuthenticated
		}
	}

	key = datastore.NewIncompleteKey(c.ctx, e.Entity.Name, nil)

	return c, key, err
}

// Gets appengine context and datastore key with optional namespace. It doesn't fail if request is not authenticated.
func (e *PreparedEntity) NewKey(c Context, nameId interface{}, withNamespace bool) (Context, *datastore.Key, error) {
	var key *datastore.Key
	var err error

	if nameId == nil {
		return c, key, ErrKeyNameIdNil
	}

	c.ctx = appengine.NewContext(c.r)

	if withNamespace {
		if c.isAuthenticated && len(c.namespace) > 0 {
			c.ctx, err = appengine.Namespace(c.ctx, c.namespace)
			if err != nil {
				return c, key, err
			}
		} else {
			err = ErrNotAuthenticated
		}
	}

	if stringId, ok := nameId.(string); ok {
		key = datastore.NewKey(c.ctx, e.Entity.Name, stringId, 0, nil)
	} else if intId, ok := nameId.(int64); ok {
		key = datastore.NewKey(c.ctx, e.Entity.Name, "", intId, nil)
	} else {
		return c, key, ErrKeyNameIdInvalidType
	}

	return c, key, err
}
