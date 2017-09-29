package sdk

import (
	"google.golang.org/appengine/datastore"
	"fmt"
)

// PreparedEntity data holder
type EntityDataHolder struct {
	Entity *Entity

	data Data // this can be edited by load/save, and conditionally with appendField functions
}

type Data map[*Field]interface{}

var (
	ErrNamedFieldNotDefined                string = "named field '%s' is not defined"
	ErrDatastoreFieldPropertyMultiDismatch string = "datastore field '%s' doesn't match in property multi"
	ErrFieldRequired                       string = "field '%s' required"
)


// Appends value
func (e *EntityDataHolder) unsafeAppendValue(name string, value interface{}) error {
	if field, ok := e.Entity.Fields[name]; ok {
		if prop.Multiple != field.Multiple {
			return fmt.Errorf(ErrDatastoreFieldPropertyMultiDismatch, prop.Name)
		}
		if field.Multiple {
			// Todo: Check if this check is necessary
			if _, ok := e.data[field]; !ok {
				e.data[field] = []interface{}{}
			}
			e.data[field] = append(e.data[field].([]interface{}), prop.Value)
		} else {
			e.data[field] = prop.Value
		}
		return nil
	} else {
		return fmt.Errorf(ErrNamedFieldNotDefined, name)
	}
}

func (e *EntityDataHolder) Load(ps []datastore.Property) error {
	e.data = map[*Field]interface{}{}
	for _, prop := range ps {
		if field, ok := e.Entity.Fields[prop.Name]; ok {
			if prop.Multiple != field.Multiple {
				return fmt.Errorf(ErrDatastoreFieldPropertyMultiDismatch, prop.Name)
			}
			if field.Multiple {
				// Todo: Check if this check is necessary
				if _, ok := e.data[field]; !ok {
					e.data[field] = []interface{}{}
				}
				e.data[field] = append(e.data[field].([]interface{}), prop.Value)
			} else {
				e.data[field] = prop.Value
			}
			return nil
		} else {
			return fmt.Errorf(ErrNamedFieldNotDefined, prop.Name)
		}
	}
	return nil
}

func (e *EntityDataHolder) Save() ([]datastore.Property, error) {
	var ps []datastore.Property

	// check if required fields are there
	for _, field := range e.Entity.requiredFields {
		if _, ok := e.data[field]; !ok {
			return ps, fmt.Errorf(ErrFieldRequired, field.Name)
		}
	}

	// create datastore property list
	for field, value := range e.data {
		if field.Multiple {
			for _, v := range value.([]interface{}) {
				ps = append(ps, datastore.Property{
					Name:     field.Name,
					Multiple: field.Multiple,
					Value:    v,
					NoIndex:  field.NoIndex,
				})
			}
		} else {
			ps = append(ps, datastore.Property{
				Name:     field.Name,
				Multiple: field.Multiple,
				Value:    value,
				NoIndex:  field.NoIndex,
			})
		}
	}

	return ps, nil
}
