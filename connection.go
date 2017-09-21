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

func isAuthorized(ctx Context, key *datastore.Key, guestRule bool, userRule bool) bool {
	if len(key.Namespace()) > 0 {
		if ctx.namespace == key.Namespace() {
			return userRule
		}
		return false
	}

	return guestRule
}

func (e *PreparedEntity) Read(ctx Context, key *datastore.Key) (datastore.PropertyList, error) {
	var ps datastore.PropertyList
	if isAuthorized(ctx, key, e.Entity.Rules[GuestRead], e.Entity.Rules[UserRead]) {
		err := datastore.Get(ctx.ctx, key, &ps)
		return ps, err
	}
	return ps, ErrNotAuthorized
}

func (e *PreparedEntity) Add(ctx Context, key *datastore.Key, ps datastore.PropertyList) (error) {
	var err error
	if isAuthorized(ctx, key, e.Entity.Rules[GuestAdd], e.Entity.Rules[UserAdd]) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.ctx, func(tc context.Context) error {
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
			key, err = datastore.Put(ctx.ctx, key, &ps)
		}
		return err
	}
	return ErrNotAuthorized
}

func (e *PreparedEntity) Edit(ctx Context, key *datastore.Key, ps datastore.PropertyList) (error) {
	var err error
	if isAuthorized(ctx, key, e.Entity.Rules[GuestEdit], e.Entity.Rules[UserEdit]) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.ctx, func(tc context.Context) error {
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
			return ErrKeyIncomplete
		}
		return err
	}
	return ErrNotAuthorized
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
