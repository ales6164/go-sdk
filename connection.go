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

func (e *Entity) Get(ctx Context, key *datastore.Key) (EntityDataHolder, error) {
	var h EntityDataHolder = e.New()
	h.isNew = false
	if isAuthorized(ctx, key, ctx.HasScope(ScopeGet)) {
		err := datastore.Get(ctx.Context, key, &h)
		if err != nil {
			return h, err
		}
		if e.OnAfterRead != nil {
			err = e.OnAfterRead(&h)
		}
		return h, err
	}
	return h, ErrNotAuthorized
}

func (e *Entity) Add(ctx Context, key *datastore.Key, h EntityDataHolder) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeAdd)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						key, err = datastore.Put(tc, key, &h)
						return err
					}
					return err
				} else {
					return EntityAlreadyExists.Params(key.Kind())
				}
				return nil
			}, nil)

		} else {
			key, err = datastore.Put(ctx.Context, key, &h)
		}
		return key, err
	}
	return key, ErrNotAuthorized
}

func (e *Entity) Put(ctx Context, key *datastore.Key, h EntityDataHolder) (*datastore.Key, error) {
	if isAuthorized(ctx, key, ctx.HasScope(ScopePut)) {
		return datastore.Put(ctx.Context, key, &h)
	}
	return key, ErrNotAuthorized
}

func (e *Entity) Edit(ctx Context, key *datastore.Key, h EntityDataHolder) (*datastore.Key, error) {
	var err error
	if isAuthorized(ctx, key, ctx.HasScope(ScopeEdit)) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					return err
				}

				key, err = datastore.Put(tc, key, &h)
				return err
			}, nil)
		} else {
			return key, ErrKeyIncomplete
		}
		return key, err
	}
	return key, ErrNotAuthorized
}
