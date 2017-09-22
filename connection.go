package sdk

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"errors"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrNotAuthorized    = errors.New("not authorized")
	ErrKeyIncomplete    = errors.New("key incomplete")
)

// don't allow user to edit other user's data
func isAuthorized(ctx Context, key *datastore.Key, hasScope bool) bool {
	if hasScope && len(key.Namespace()) != 0 {
		if ctx.Namespace == key.Namespace() {
			return true
		}
		return false
	}

	return hasScope
}

func (e *PreparedEntity) Get(ctx Context, key *datastore.Key) (datastore.PropertyList, error) {
	var ps datastore.PropertyList
	if isAuthorized(ctx, key, ctx.HasScope(ScopeGet)) {
		err := datastore.Get(ctx.Context, key, &ps)
		return ps, err
	}
	return ps, ErrNotAuthorized
}

func (e *PreparedEntity) Add(ctx Context, key *datastore.Key, ps datastore.PropertyList) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeAdd)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						key, err = datastore.Put(tc, key, &ps)
						return err
					}
					return err
				} else {
					return EntityAlreadyExists.Params(key.Kind())
				}
				return nil
			}, nil)

		} else {
			key, err = datastore.Put(ctx.Context, key, &ps)
		}
		return key, err
	}
	return key, ErrNotAuthorized
}

func (e *PreparedEntity) Put(ctx Context, key *datastore.Key, ps datastore.PropertyList) (*datastore.Key, error) {
	if isAuthorized(ctx, key, ctx.HasScope(ScopePut)) {
		return datastore.Put(ctx.Context, key, &ps)
	}
	return key, ErrNotAuthorized
}

func (e *PreparedEntity) Edit(ctx Context, key *datastore.Key, ps datastore.PropertyList) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeEdit)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						key, err = datastore.Put(tc, key, &ps)
						return err
					}
					return err
				} else {
					return EntityAlreadyExists.Params(key.Kind())
				}
				return nil
			}, nil)
		} else {
			return key, ErrKeyIncomplete
		}
		return key, err
	}
	return key, ErrNotAuthorized
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
						data[field.GroupName] = map[string]interface{}{
							"LastPropCount": 0,
							"LastProp":      "",
							"count":         0,
							"items":         []map[string]interface{}{},
						}
					}

					var groupField map[string]interface{} = data[field.GroupName].(map[string]interface{})

					if groupField["LastProp"] != name {
						groupField["LastPropCount"] = 0
						groupField["LastProp"] = name
					} else {
						groupField["LastPropCount"] = groupField["LastPropCount"].(int) + 1
					}

					if len(groupField["items"].([]map[string]interface{}))-1 < groupField["LastPropCount"].(int) {
						groupField["items"] = append(groupField["items"].([]map[string]interface{}), map[string]interface{}{})
					}

					groupField["items"].([]map[string]interface{})[groupField["LastPropCount"].(int)][name] = prop.Value
					groupField["count"] = len(groupField["items"].([]map[string]interface{}))

					/*delete(groupField, "LastPropCount")
					delete(groupField, "LastProp")*/

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

func (e *PreparedEntity) GetData(list datastore.PropertyList) (map[string]interface{}, interface{}) {
	var data = map[string]interface{}{}
	var err interface{}

	for _, prop := range list {
		if field, ok := e.Entity.Fields[prop.Name]; ok {
			if field.Json == NoJsonOutput {
				continue
			}

			var name = string(field.Json)
			if len(name) == 0 {
				name = field.Name
			}
			var isGrouped bool
			if len(field.GroupName) != 0 {
				isGrouped = true
			}

			if field.Multiple {
				if isGrouped {

					if _, ok := data[field.GroupName]; !ok {
						data[field.GroupName] = map[string]interface{}{
							"LastPropCount": 0,
							"LastProp":      "",
							"count":         0,
							"items":         []map[string]interface{}{},
						}
					}

					var groupField map[string]interface{} = data[field.GroupName].(map[string]interface{})

					if groupField["LastProp"] != name {
						groupField["LastPropCount"] = 0
						groupField["LastProp"] = name
					} else {
						groupField["LastPropCount"] = groupField["LastPropCount"].(int) + 1
					}

					if len(groupField["items"].([]map[string]interface{}))-1 < groupField["LastPropCount"].(int) {
						groupField["items"] = append(groupField["items"].([]map[string]interface{}), map[string]interface{}{})
					}

					groupField["items"].([]map[string]interface{})[groupField["LastPropCount"].(int)][name] = prop.Value
					groupField["count"] = len(groupField["items"].([]map[string]interface{}))

					/*delete(groupField, "LastPropCount")
					delete(groupField, "LastProp")*/

					data[field.GroupName] = groupField
				} else {
					if _, ok := data[name]; !ok {
						data[name] = []interface{}{}
					}
					if _, ok := data[name].([]interface{}); !ok {
						return data, name
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

	return data, err
}
