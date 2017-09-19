package sdk

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	gcontext"github.com/gorilla/context"
	"google.golang.org/appengine"
	"regexp"
	"errors"
)

type PreparedEntity struct {
	Input          map[string]interface{}
	Ready          map[*Field]interface{}
	Output         datastore.PropertyList
	Key            *PreparedKey
	ParentKey      *PreparedKey
	RequiredFields []string
	Entity         *Ent
}

type PreparedKey struct {
	HasParent           bool
	Complete            bool
	IsRequiredFromInput bool

	FromToken bool
	FromField string

	NamespaceType
	Kind     string
	StringID string
	IntID    int64
}

type DataObject struct {
	Input  map[string]interface{}
	Output datastore.PropertyList
}

// Fills static data and assembles keys
func PrepareEntity(ent *Ent) *PreparedEntity {
	var prepared = new(PreparedEntity)
	prepared.Entity = ent
	prepared.Ready = map[*Field]interface{}{}
	prepared.Input = map[string]interface{}{}

	var keyRequiresFieldInput bool
	var keyParentRequiresFieldInput bool

	prepared.Key = new(PreparedKey)
	if ent.Key != nil {
		prepared.Key.FromField = ent.Key.FromField
		prepared.Key.FromToken = ent.Key.FromToken
		prepared.Key.Kind = ent.Key.Kind
		prepared.Key.StringID = ent.Key.StringID
		prepared.Key.IntID = ent.Key.IntID
		prepared.Key.NamespaceType = ent.Key.NamespaceType

		if len(prepared.Key.FromField) > 0 {
			keyRequiresFieldInput = true
			prepared.Key.IsRequiredFromInput = true
		} else {
			prepared.Key.Complete = true
		}
	}

	if len(prepared.Key.Kind) == 0 {
		prepared.Key.Kind = ent.Name
	}

	if ent.ParentKey != nil {
		prepared.ParentKey = new(PreparedKey)

		prepared.ParentKey.FromField = ent.ParentKey.FromField
		prepared.ParentKey.FromToken = ent.ParentKey.FromToken
		prepared.ParentKey.Kind = ent.ParentKey.Kind
		prepared.ParentKey.StringID = ent.ParentKey.StringID
		prepared.ParentKey.IntID = ent.ParentKey.IntID
		prepared.ParentKey.NamespaceType = ent.ParentKey.NamespaceType

		if len(prepared.ParentKey.FromField) > 0 {
			keyParentRequiresFieldInput = true
			prepared.ParentKey.IsRequiredFromInput = true
		} else {
			prepared.ParentKey.Complete = true
		}

		prepared.Key.HasParent = true
	}

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

		if keyRequiresFieldInput && ent.Key.FromField == fieldName {
			if field.WithStaticValue != nil {
				if stringStaticValue, ok := field.WithStaticValue.(string); ok {
					prepared.Key.StringID = stringStaticValue
				} else if int64StaticValue, ok := field.WithStaticValue.(int64); ok {
					prepared.Key.IntID = int64StaticValue
				} else {
					panic(errors.New("static value key id is of invalid type"))
				}
				prepared.Key.Complete = true
				prepared.Key.IsRequiredFromInput = false
				keyRequiresFieldInput = false
			}
		}

		if keyParentRequiresFieldInput && ent.ParentKey.FromField == fieldName {
			if field.WithStaticValue != nil {
				if stringStaticValue, ok := field.WithStaticValue.(string); ok {
					prepared.ParentKey.StringID = stringStaticValue
				} else if int64StaticValue, ok := field.WithStaticValue.(int64); ok {
					prepared.ParentKey.IntID = int64StaticValue
				} else {
					panic(errors.New("static value key id is of invalid type"))
				}
				prepared.ParentKey.Complete = true
				prepared.ParentKey.IsRequiredFromInput = false
				keyParentRequiresFieldInput = false
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

func (e *PreparedEntity) GetOutputData(list datastore.PropertyList) map[string]interface{} {
	var data = map[string]interface{}{}
	for _, prop := range list {
		if field, ok := e.Entity.Fields[prop.Name]; ok {
			if field.Json == NoJsonOutput {
				continue
			}

			var name = string(field.Json)

			if len(field.GroupName) == 0 {
				if field.Multiple {
					if _, ok := data[name]; !ok {
						data[name] = []interface{}{}
					}
					data[name] = append(data[name].([]interface{}), prop.Value)
				} else {
					data[name] = prop.Value
				}
			} else {
				if _, ok := data[field.GroupName]; !ok {
					data[field.GroupName] = map[string]interface{}{}
				}

				if field.Multiple {
					if _, ok := data[field.GroupName].(map[string]interface{})[name]; !ok {
						data[field.GroupName].(map[string]interface{})[name] = []interface{}{}
					}
					data[field.GroupName].(map[string]interface{})[name] = append(data[field.GroupName].(map[string]interface{})[name].([]interface{}), prop.Value)
				} else {
					data[field.GroupName].(map[string]interface{})[name] = prop.Value
				}
			}
		}
	}
	return data
}

type GroupField struct {
	Count         int                      `json:"count"`
	LastPropCount int                      `json:"-"`
	LastProp      string                   `json:"-"`
	Items         []map[string]interface{} `json:"items"`
}

func (e *PreparedEntity) GetGroupedOutputData(list datastore.PropertyList) map[string]interface{} {
	var data = map[string]interface{}{}

	for _, prop := range list {
		if field, ok := e.Entity.Fields[prop.Name]; ok {
			if field.Json == NoJsonOutput {
				continue
			}

			var name = string(field.Json)
			var isGrouped bool
			if len(field.GroupName) != 0 {
				isGrouped = true
			}

			if field.Multiple {
				if isGrouped {

					if _, ok := data[field.GroupName]; !ok {
						data[field.GroupName] = GroupField{
							LastPropCount: 0,
						}
					}

					var groupField GroupField = data[field.GroupName].(GroupField)

					if groupField.LastProp != name {
						groupField.LastPropCount = 0
						groupField.LastProp = name
					} else {
						groupField.LastPropCount += 1
					}

					if len(groupField.Items) - 1 < groupField.LastPropCount {
						groupField.Items = append(groupField.Items, map[string]interface{}{})
					}

					groupField.Items[groupField.LastPropCount][name] = prop.Value
					groupField.Count = len(groupField.Items)

					data[field.GroupName] = groupField
				} else {
					if _, ok := data[name]; !ok {
						data[name] = []interface{}{}
					}
					data[name] = append(data[name].([]interface{}), prop.Value)
				}

			} else {
				if isGrouped {
					if _, ok := data[field.GroupName]; !ok {
						data[field.GroupName] = map[string]interface{}{}
					}

					data[field.GroupName].(map[string]interface{})[name] = prop.Value
				} else {
					data[name] = prop.Value
				}

			}
		}
	}

	return data
}

func (k *PreparedKey) GetKey(ctx context.Context, kind string, parent *datastore.Key) *datastore.Key {
	if k != nil {
		if k.Complete {
			return datastore.NewKey(ctx, kind, k.StringID, k.IntID, parent)
		}
		return datastore.NewIncompleteKey(ctx, kind, parent)
	}
	return nil
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

func (e *PreparedEntity) FromMap(c Context, dataMap map[string]interface{}, checkIfRequired bool) (context.Context, *datastore.Key, *DataObject, error) {
	var ctx context.Context
	var key *datastore.Key
	var err error
	var dataObject = &DataObject{
		Input:  e.Input,
		Output: e.Output,
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
					return ctx, key, dataObject, err
				}
			}
		} else {
			err = e.rangeOverData(name, values, dataObject)
			if err != nil {
				return ctx, key, dataObject, err
			}
		}
	}

	ctx, key, err = e.prepare(c, dataObject, checkIfRequired)
	return ctx, key, dataObject, err

}

func (e *PreparedEntity) FromForm(c Context, checkIfRequired bool) (context.Context, *datastore.Key, *DataObject, error) {
	var ctx context.Context
	var key *datastore.Key
	var err error
	var dataObject = &DataObject{
		Input:  e.Input,
		Output: e.Output,
	}

	// append ready fields
	e.appendReadyFields(dataObject)

	//todo: fix this
	c.r.FormValue("a")

	if err = c.r.ParseForm(); err != nil {
		return ctx, key, dataObject, err
	}

	for name, values := range c.r.Form {

		// remove '[]' from fieldName if it's an array
		if len(name) > 2 && name[len(name)-2:] == "[]" {
			name = name[:len(name)-2]
		}

		for _, v := range values {
			err = e.rangeOverData(name, v, dataObject)
			if err != nil {
				return ctx, key, dataObject, err
			}
		}

	}

	ctx, key, err = e.prepare(c, dataObject, checkIfRequired)
	return ctx, key, dataObject, err

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

func (e *PreparedEntity) prepare(c Context, dataObj *DataObject, check bool) (context.Context, *datastore.Key, error) {
	var ctx context.Context
	var key *datastore.Key
	var err error

	// check if it has all required fields
	if check {
		for _, fieldName := range e.RequiredFields {
			if _, ok := dataObj.Input[fieldName]; !ok {
				return ctx, key, FieldRequired.Params(fieldName)
			}
		}
	}

	ctx = appengine.NewContext(c.r)

	// set request token
	var hasResolvedToken bool
	var hasAuthorizationErr bool
	var username string
	var getUsername = func() (string, error) {
		if !hasResolvedToken && !hasAuthorizationErr {
			requestToken := gcontext.Get(c.r, "user")
			if requestToken != nil {
				var ok bool
				username, ok = resolveToken(requestToken)
				if ok {
					hasResolvedToken = true
					return username, nil
				}
				return username, Unauthorized.Params("invalid token")
			}
		}
		hasAuthorizationErr = true
		return username, GuestAccessRequest
	}
	var getNamespaceContext = func(ns NamespaceType) (context.Context, error) {
		switch ns {
		case UserNamespace:
			u, err := getUsername()
			if err != nil {
				return ctx, err
			}

			appCtx, err := appengine.Namespace(ctx, u)
			if err != nil {
				return ctx, err
			}

			return appCtx, Unauthorized.Params("invalid token")
		case NoNamespace:
			return ctx, nil
		}
		return ctx, InvalidNamespaceType.Params(ns)
	}

	// create keys
	var parentKey *datastore.Key
	if e.ParentKey != nil {
		parentKeyNsCtx, err := getNamespaceContext(e.ParentKey.NamespaceType)
		if err != nil {
			return ctx, key, err
		}
		if e.ParentKey.FromToken {
			u, err := getUsername()
			if err != nil && err != GuestAccessRequest {
				return ctx, key, err
			}
			parentKey = datastore.NewKey(parentKeyNsCtx, e.ParentKey.Kind, u, 0, nil)
		}
		if e.ParentKey.IsRequiredFromInput {
			if probablyKeyId, ok := dataObj.Input[e.ParentKey.FromField]; ok {
				// check if is string or int64
				if _, ok := probablyKeyId.(string); ok {
					parentKey = datastore.NewKey(parentKeyNsCtx, e.ParentKey.Kind, probablyKeyId.(string), 0, nil)
				} else if _, ok := probablyKeyId.(int64); ok {
					parentKey = datastore.NewKey(parentKeyNsCtx, e.ParentKey.Kind, "", probablyKeyId.(int64), nil)
				} else {
					return ctx, key, IdFieldValueTypeError.Params(e.ParentKey.FromField)
				}
			} else {
				return ctx, key, NoIdFieldValue.Params(e.ParentKey.FromField)
			}
		} else {
			parentKey = e.ParentKey.GetKey(parentKeyNsCtx, e.ParentKey.Kind, nil)
		}
	}

	ctx, err = getNamespaceContext(e.Key.NamespaceType)
	if err != nil {
		return ctx, key, err
	}
	if e.Key.FromToken {
		u, err := getUsername()
		if err != nil && err != GuestAccessRequest {
			return ctx, key, err
		}
		key = datastore.NewKey(ctx, e.Key.Kind, u, 0, parentKey)
	}
	if e.Key.IsRequiredFromInput {
		//todo:
		if probablyKeyId, ok := dataObj.Input[e.Key.FromField]; ok {
			// check if is string or int64
			if _, ok := probablyKeyId.(string); ok {
				key = datastore.NewKey(ctx, e.Key.Kind, probablyKeyId.(string), 0, parentKey)
			} else if _, ok := probablyKeyId.(int64); ok {
				key = datastore.NewKey(ctx, e.Key.Kind, "", probablyKeyId.(int64), parentKey)
			} else {
				return ctx, key, IdFieldValueTypeError.Params(e.Key.FromField)
			}
		} else {
			return ctx, key, NoIdFieldValue.Params(e.Key.FromField)
		}
	} else {
		key = e.ParentKey.GetKey(ctx, e.Key.Kind, parentKey)
	}

	return ctx, key, nil
}
