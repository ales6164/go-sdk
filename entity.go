package sdk

import "google.golang.org/appengine/datastore"

type Entity struct {
	Name   string
	Fields map[string]*Field

	preparedData Data

	requiredFields []*Field

	// listeners
	OnAfterRead func(data map[string]interface{}, list *datastore.PropertyList) (map[string]interface{}, error)
}

func NewEntity(name string, fields []*Field) *Entity {
	var e = new(Entity)
	e.Name = name

	e.preparedData = Data{}

	e.Fields = map[string]*Field{}
	for _, field := range fields {
		e.Fields[field.Name] = field

		if field.IsRequired {
			e.requiredFields = append(e.requiredFields, field)
		}

		if field.DefaultValue != nil {
			e.preparedData[field] = field.DefaultValue
		}

		if field.StaticValue != nil {
			e.preparedData[field] = field.StaticValue
		}

		if field.ValueFunc != nil {
			field.fieldFunc = append(field.fieldFunc, func(v interface{}) (interface{}, error) {
				return field.ValueFunc(), nil
			})
		}

		if field.TransformFunc != nil {
			field.fieldFunc = append(field.fieldFunc, field.TransformFunc)
		}
	}

	return e
}

func (e *Entity) New() EntityDataHolder {
	var dataHolder = EntityDataHolder{
		Entity: e,
		data:   Data{},
	}

	// copy prepared values
	for name, value := range e.preparedData {
		dataHolder.data[name] = value
	}

	return dataHolder
}
