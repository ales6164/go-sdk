package sdk

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"errors"
	"fmt"
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

type EntityQueryFilter struct {
	Name     string      `json:"name"`
	Operator string      `json:"operator"` // =, <, <=, >, >=
	Value    interface{} `json:"value"`
}

func (e *Entity) Query(ctx Context, namespace string, sort string, limit int, filters ...EntityQueryFilter) ([]*EntityDataHolder, error) {
	var hs []*EntityDataHolder

	if ctx.HasScope(ScopeGet) && (len(namespace) == 0 || ctx.Namespace == namespace) {
		q := datastore.NewQuery(e.Name)

		for _, filter := range filters {
			q = q.Filter(fmt.Sprintf("%s %s", filter.Name, filter.Operator), filter.Value)
		}

		if len(sort) != 0 {
			q = q.Order(sort)
		}

		if limit != 0 {
			q = q.Limit(limit)
		}

		t := q.Run(ctx.Context)
		for {
			var h *EntityDataHolder = e.New()
			h.isNew = false
			_, err := t.Next(h)
			if err == datastore.Done {
				break
			}
			if err != nil {
				return hs, err
			}
			if e.OnAfterRead != nil {
				err = e.OnAfterRead(h)
			}
			hs = append(hs, h)
		}

		return hs, nil
	}

	return hs, ErrNotAuthorized
}

func (e *Entity) Get(ctx Context, key *datastore.Key) (*EntityDataHolder, error) {
	var h *EntityDataHolder = e.New()
	h.isNew = false
	if isAuthorized(ctx, key, ctx.HasScope(ScopeGet)) {
		err := datastore.Get(ctx.Context, key, h)
		if err != nil {
			return h, err
		}
		if e.OnAfterRead != nil {
			err = e.OnAfterRead(h)
		}
		return h, err
	}
	return h, ErrNotAuthorized
}

func (e *Entity) Add(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeAdd)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						key, err = datastore.Put(tc, key, h)
						e.PutToIndexes(tc, key.Encode(), h)
						return err
					}
					return err
				} else {
					return EntityAlreadyExists.Params(key.Kind())
				}
				return nil
			}, nil)

		} else {
			key, err = datastore.Put(ctx.Context, key, h)
			e.PutToIndexes(ctx.Context, key.Encode(), h)
		}
		return key, err
	}
	return key, ErrNotAuthorized
}

func (e *Entity) Put(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	if isAuthorized(ctx, key, ctx.HasScope(ScopePut)) {
		e.PutToIndexes(ctx.Context, key.Encode(), h)
		return datastore.Put(ctx.Context, key, h)
	}
	return key, ErrNotAuthorized
}

func (e *Entity) Edit(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeEdit)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					return err
				}

				key, err = datastore.Put(tc, key, h)
				return err
			}, nil)
		} else {
			return key, ErrKeyIncomplete
		}
		return key, err
	}
	return key, ErrNotAuthorized
}
