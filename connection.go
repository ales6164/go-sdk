package sdk

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrNotAuthorized    = errors.New("not authorized")
	ErrKeyIncomplete    = errors.New("key incomplete")
)

type EntityQueryFilter struct {
	Name     string      `json:"name"`
	Operator string      `json:"operator"` // =, <, <=, >, >=
	Value    interface{} `json:"value"`
}

func (e *Entity) Query(ctx Context, sort string, limit int, filters ...EntityQueryFilter) ([]*EntityDataHolder, error) {
	var hs []*EntityDataHolder

	if ctx.HasScope(e.Name, ScopeRead) {
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
			var h *EntityDataHolder = e.New(ctx)
			h.isNew = false
			key, err := t.Next(h)
			if err == datastore.Done {
				break
			}
			if err != nil {
				return hs, err
			}
			h.id = key.Encode()

			if e.Private {
				if !ctx.UserMatches(h.Get("_createdBy")) {
					continue
				}
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
	var h *EntityDataHolder = e.New(ctx)
	h.isNew = false
	if ctx.HasScope(e.Name, ScopeRead) {
		err := datastore.Get(ctx.Context, key, h)
		if err != nil {
			return h, err
		}
		encoded := key.Encode()
		h.id = encoded
		if e.Private {
			if !ctx.UserMatches(h.Get("_createdBy")) {
				return nil, ErrNotAuthorized
			}
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
	if ctx.HasScope(e.Name, ScopeAdd) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					if err == datastore.ErrNoSuchEntity {
						key, err = datastore.Put(tc, key, h)
						if err != nil {
							return err
						}
						encoded := key.Encode()
						h.id = encoded
						e.PutToIndexes(tc, encoded, h)
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
			if err != nil {
				return key, err
			}
			encoded := key.Encode()
			h.id = encoded
			e.PutToIndexes(ctx.Context, encoded, h)
		}
		return key, err
	}
	return key, ErrNotAuthorized
}

func (e *Entity) Put(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	if ctx.HasScope(e.Name, ScopeWrite) {
		key, err := datastore.Put(ctx.Context, key, h)
		if err != nil {
			return key, err
		}
		encoded := key.Encode()
		h.id = encoded
		if e.Private {
			if !ctx.UserMatches(h.Get("_createdBy")) {
				return nil, ErrNotAuthorized
			}
		}
		e.PutToIndexes(ctx.Context, encoded, h)
		return key, err
	}
	return key, ErrNotAuthorized
}

// Checks if the key is incomplete; if not it tries retrieving the entity and then copying values to the existing entity
// otherwise it adds a new entity
func (e *Entity) Post(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	var err error

	if key.Incomplete() {
		if ctx.HasScope(e.Name, ScopeAdd) {
			// add entity

			key, err = datastore.Put(ctx.Context, key, h)
			if err != nil {
				return key, err
			}
			encoded := key.Encode()
			h.id = encoded
			if e.Private {
				if !ctx.UserMatches(h.Get("_createdBy")) {
					return nil, ErrNotAuthorized
				}
			}
			e.PutToIndexes(ctx.Context, encoded, h)

			return key, err
		}
	} else if ctx.HasScope(e.Name, ScopeEdit) || ctx.HasScope(e.Name, ScopeWrite) {
		// edit or rewrite entity
		err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {

			h.keepExistingValue = true // important!
			err := datastore.Get(tc, key, h)
			if err == nil {
				if e.Private {
					if !ctx.UserMatches(h.Get("_createdBy")) {
						return ErrNotAuthorized
					}
				}
			} else {
				if err == datastore.ErrNoSuchEntity {
					// Add entity

					key, err = datastore.Put(ctx.Context, key, h)
					if err != nil {
						return err
					}
					encoded := key.Encode()
					h.id = encoded
					e.PutToIndexes(ctx.Context, encoded, h)

					return err
				}
				return err
			}
			h.keepExistingValue = false

			key, err = datastore.Put(ctx.Context, key, h)
			if err != nil {
				return err
			}
			encoded := key.Encode()
			h.id = encoded
			e.PutToIndexes(ctx.Context, encoded, h)

			return err
		}, nil)

		return key, err
	}

	return key, ErrNotAuthorized
}

// currently it only check if entity exists and rewrites it
func (e *Entity) Edit(ctx Context, key *datastore.Key, h *EntityDataHolder) (*datastore.Key, error) {
	var err error
	if ctx.HasScope(e.Name, ScopeEdit) {
		if !key.Incomplete() {
			err = datastore.RunInTransaction(ctx.Context, func(tc context.Context) error {
				var tempEnt datastore.PropertyList
				err := datastore.Get(tc, key, &tempEnt)
				if err != nil {
					return err
				}
				if e.Private {
					if !ctx.UserMatches(h.Get("_createdBy")) {
						return ErrNotAuthorized
					}
				}

				key, err = datastore.Put(tc, key, h)
				if err != nil {
					return err
				}
				encoded := key.Encode()
				h.id = encoded
				return err
			}, nil)
		} else {
			return key, ErrKeyIncomplete
		}
		return key, err
	}
	return key, ErrNotAuthorized
}
