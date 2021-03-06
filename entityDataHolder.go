package sdk

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
)

// PreparedEntity data holder
type EntityDataHolder struct {
	Entity *Entity `json:"-"`

	isNew             bool
	keepExistingValue bool // turn this true when receiving old data from database; used for editing existing entity

	Id    string                 // saved during datastore operations and returned on output
	data  Data                   // this can be edited by load/save, and conditionally with appendField functions
	input map[string]interface{} // this can be edited by load/save, and conditionally with appendField functions
}

type Data map[*Field]interface{}

const (
	ErrNamedFieldNotDefined                string = "named field '%s' is not defined"
	ErrDatastoreFieldPropertyMultiDismatch string = "datastore field '%s' doesn't match in property multi"
	ErrFieldRequired                       string = "field '%s' required"
	ErrFieldEditPermissionDenied           string = "field '%s' edit permission denied"
	ErrFieldValueNotValid                  string = "field '%s' value is not valid"
	ErrFieldValueTypeNotValid              string = "field '%s' value type is not valid"
	ErrValueIsNil                          string = "field '%s' value is empty"
)

func init() {
	gob.Register(time.Now())
	gob.Register(datastore.Key{})
}

func (e *EntityDataHolder) Get(ctx Context, name string) interface{} {
	var endValue interface{}

	sep := strings.Split(name, ".")
	if field, ok := e.Entity.fields[sep[0]]; ok {
		var value = e.data[field]

		if field.Lookup && len(field.Entity) > 0 {
			if field.Multiple {
				endValue = []interface{}{}

				for _, v := range value.([]interface{}) {
					endV, _ := Entities[field.Entity].Lookup(ctx, v.(string))
					endValue = append(endValue.([]interface{}), endV)
				}
			} else {
				endValue, _ = Entities[field.Entity].Lookup(ctx, value.(string))
			}
		} else {
			endValue = value
		}

		if len(sep) > 1 {
			for i := 1; i < len(sep); i++ {
				if endMap, ok := endValue.(map[string]interface{}); ok {
					endValue = endMap[sep[i]]
					/*if endValue, ok = endMap[sep[i]]; ok {}*/
				} else {
					return nil
				}
			}
		}

		return endValue
	}
	return nil
}

func (e *EntityDataHolder) GetInput(name string) interface{} {
	return e.input[name]
}

func output(ctx Context, id string, data Data, cacheLookup bool) map[string]interface{} {
	var output = map[string]interface{}{}
	var multiples []string

	// range over data. Value can be single value or if the field it Multiple then it's an array
	for field, value := range data {
		var doCacheLookup = cacheLookup && field.Lookup && len(field.Entity) > 0

		if field.Json == NoJsonOutput {
			continue
		}

		if len(field.GroupName) != 0 {
			if _, ok := output[field.GroupName]; !ok {
				output[field.GroupName] = map[string]interface{}{}
			}

			if field.Multiple {
				for _, v := range value.([]interface{}) {

					if _, ok := output[field.GroupName].(map[string]interface{})["items"]; !ok {
						output[field.GroupName] = map[string]interface{}{
							"LastPropCount": 0,
							"LastProp":      "",
							"count":         0,
							"items":         []map[string]interface{}{},
						}
						multiples = append(multiples, field.GroupName)
					}

					var groupField = output[field.GroupName].(map[string]interface{})

					if groupField["LastProp"] != field.Name {
						groupField["LastPropCount"] = 0
						groupField["LastProp"] = field.Name
					} else {
						groupField["LastPropCount"] = groupField["LastPropCount"].(int) + 1
					}

					if len(groupField["items"].([]map[string]interface{}))-1 < groupField["LastPropCount"].(int) {
						groupField["items"] = append(groupField["items"].([]map[string]interface{}), map[string]interface{}{})
					}

					if doCacheLookup {
						v, _ = Entities[field.Entity].Lookup(ctx, v.(string))
					}

					groupField["items"].([]map[string]interface{})[groupField["LastPropCount"].(int)][field.Name] = v
					groupField["count"] = len(groupField["items"].([]map[string]interface{}))

					output[field.GroupName] = groupField

				}
			} else {
				if doCacheLookup {
					value, _ = Entities[field.Entity].Lookup(ctx, value.(string))
				}

				output[field.GroupName].(map[string]interface{})[field.Name] = value
			}
		} else {
			if doCacheLookup {
				value, _ = Entities[field.Entity].Lookup(ctx, value.(string))
			}

			output[field.Name] = value
		}
	}

	for _, multiName := range multiples {
		delete(output[multiName].(map[string]interface{}), "LastPropCount")
		delete(output[multiName].(map[string]interface{}), "LastProp")
	}

	output["_id"] = id

	return output
}

func flatOutput(id string, data Data) map[string]interface{} {
	var output = map[string]interface{}{}

	for field, value := range data {
		if field.Json == NoJsonOutput {
			continue
		}

		if len(field.GroupName) != 0 {
			output[field.GroupName+strings.Title(field.Name)] = value
		} else {
			output[field.Name] = value
		}
	}

	output["_id"] = id

	return output
}

func (e *EntityDataHolder) Output(ctx Context) map[string]interface{} {
	return output(ctx, e.Id, e.data, true)
}

func (e *EntityDataHolder) FlatOutput() map[string]interface{} {
	return flatOutput(e.Id, e.data)
}

func (e *EntityDataHolder) JSON(ctx Context) (string, error) {
	bs, err := json.Marshal(e.Output(ctx))
	return string(bs), err
}

// Safely appends value
func (e *EntityDataHolder) AppendValue(name string, value interface{}) error {
	if field, ok := e.Entity.fields[name]; ok {
		var c = &ValueContext{Field: field, Trust: Base}
		return e.appendFieldValue(field, value, c)
	}

	// skip
	//return fmt.Errorf(ErrNamedFieldNotDefined, name)

	return nil
}

func (e *EntityDataHolder) appendValue(name string, value interface{}, trust ValueTrust) error {
	e.input[name] = value

	if field, ok := e.Entity.fields[name]; ok {

		// to keep it from deleting value
		// todo
		if (field.Type == FileType || field.Type == ImageType) && field.IsRequired {
			if fileUrl, ok := value.(string); !ok || len(fileUrl) == 0 {
				return nil
			}
		}

		var c = &ValueContext{Field: field, Trust: trust}
		return e.appendFieldValue(field, value, c)
	}

	// skip
	//return fmt.Errorf(ErrNamedFieldNotDefined, name)

	return nil
}

// Safely appends value
func (e *EntityDataHolder) appendFieldValue(field *Field, value interface{}, vc *ValueContext) error {
	if !e.isNew && field.NoEdits {
		return fmt.Errorf(ErrFieldEditPermissionDenied, field.datastoreFieldName)
	}

	var v = value
	var err error
	for _, fun := range field.fieldFunc {
		v, err = fun(vc, v)
		if err != nil {
			return err
		}
	}

	if v != nil {
		e.unsafeAppendFieldValue(field, v, value, e.keepExistingValue)
		return nil
	}

	return fmt.Errorf(ErrValueIsNil, field.datastoreFieldName)
}

// UNSAFE Appends value without any checks
func (e *EntityDataHolder) unsafeAppendFieldValue(field *Field, value interface{}, formValue interface{}, keepExistingValue bool) {
	if field.Multiple {
		// Todo: Check if this check is necessary
		if _, ok := e.data[field]; !ok {
			e.data[field] = []interface{}{}
		} else if keepExistingValue {
			return
		}
		if _, ok := e.data[field].([]interface{}); !ok {
			panic(errors.New("field '" + field.Name + "' value is not []interface{}"))
		}
		e.data[field] = append(e.data[field].([]interface{}), value)
	} else {
		if _, ok := e.data[field]; ok && keepExistingValue {
			return
		}
		e.data[field] = value
	}
}

// load from datastore properties into Data map
func (e *EntityDataHolder) Load(ps []datastore.Property) error {
	/*e.data = map[*Field]interface{}{}*/
	for _, prop := range ps {
		if field, ok := e.Entity.fields[prop.Name]; ok {

			if prop.Multiple != field.Multiple {
				return fmt.Errorf(ErrDatastoreFieldPropertyMultiDismatch, prop.Name)
			}
			e.unsafeAppendFieldValue(field, prop.Value, nil, e.keepExistingValue)
		} else {
			return fmt.Errorf(ErrNamedFieldNotDefined, prop.Name)
		}
	}
	return nil
}

// load Data map into datastore Property array
func (e *EntityDataHolder) Save() ([]datastore.Property, error) {
	var ps []datastore.Property

	// check if required fields are there
	for _, field := range e.Entity.requiredFields {
		if _, ok := e.data[field]; !ok {
			return ps, fmt.Errorf(ErrFieldRequired, field.datastoreFieldName)
		}
	}

	// create datastore property list
	for field, value := range e.data {
		// set group name

		if field.Multiple {
			for _, v := range value.([]interface{}) {
				ps = append(ps, datastore.Property{
					Name:     field.datastoreFieldName,
					Multiple: field.Multiple,
					Value:    v,
					NoIndex:  field.NoIndex,
				})
			}
		} else {
			ps = append(ps, datastore.Property{
				Name:     field.datastoreFieldName,
				Multiple: field.Multiple,
				Value:    value,
				NoIndex:  field.NoIndex,
			})
		}
	}

	return ps, nil
}
