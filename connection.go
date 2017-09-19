package sdk

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Connection struct {
	Ctx    context.Context
	Entity *Entity
	Name   string

	Stored      datastore.PropertyList
	MultiStored []datastore.PropertyList

	Key       *datastore.Key
	ParentKey *datastore.Key

	err error
}

type Stored struct {
	Data []map[string]interface{} `json:"-"`
}

func (s *Stored) GetData(e *Entity) []map[string]interface{} {
	var d []map[string]interface{}

	for _, stored := range s.Data {
		var data = map[string]interface{}{}
		for _, prop := range e.Properties {
			if !prop.IsPrivate {
				data[prop.Name] = stored[prop.Name]
			}
		}
		d = append(d, data)
	}

	return d
}

func (c *Connection) Parent(stringID string, intID int64, parent *datastore.Key) *Connection {
	c.ParentKey = datastore.NewKey(c.Ctx, c.Name, stringID, intID, parent)
	return c
}

func (c *Connection) Get(stringID string, intID int64) (*Stored, error) {
	var err error
	var data map[string]interface{}
	var stored = new(Stored)

	var key = datastore.NewKey(c.Ctx, c.Name, stringID, intID, c.ParentKey)

	var entData datastore.PropertyList
	if err = datastore.Get(c.Ctx, key, &entData); err != nil {
		return stored, err
	}

	data = map[string]interface{}{}
	for _, prop := range entData {
		data[prop.Name] = prop.Value
	}
	stored.Data = append(stored.Data, data)

	return stored, err
}

func Get(ctx context.Context, key *datastore.Key, list *datastore.PropertyList) (error) {

	//var data = map[string]interface{}{}

	err := datastore.Get(ctx, key, list)



	return  err
}

func (c *Connection) Post(inData map[string]interface{}) (*Stored, bool, error) {
	var stored *Stored
	var alreadyExists bool = true
	var err error

	if c.err != nil {
		return stored, alreadyExists, c.err
	}

	var key *datastore.Key
	var assembled datastore.PropertyList

	key, assembled, stored, err = assembleEntity(c.Entity, c.Ctx, inData)
	if err != nil {
		return stored, alreadyExists, err
	}

	err = datastore.RunInTransaction(c.Ctx, func(tc context.Context) error {
		var tempEnt datastore.PropertyList
		err := datastore.Get(tc, key, &tempEnt)
		if err != nil {
			if err == datastore.ErrNoSuchEntity {
				_, err := datastore.Put(tc, key, &assembled)
				alreadyExists = false
				return err
			}
			return err
		}
		return nil
	}, nil)

	return stored, alreadyExists, err
}

func Post(ctx context.Context, key *datastore.Key, list datastore.PropertyList) (*datastore.Key, error) {
	var err error
	if !key.Incomplete() {
		err = datastore.RunInTransaction(ctx, func(tc context.Context) error {
			var tempEnt datastore.PropertyList
			err := datastore.Get(tc, key, &tempEnt)
			if err != nil {
				if err == datastore.ErrNoSuchEntity {
					key, err = datastore.Put(tc, key, &list)
					return err
				}
				return err
			} else {
				return EntityAlreadyExists.Params(key.Kind())
			}
			return nil
		}, nil)

	} else {
		key, err = datastore.Put(ctx, key, &list)
	}

	return key, err
}

func (c *Connection) Put(inData map[string]interface{}) (*Stored, error) {
	var stored = new(Stored)
	var err error

	if c.err != nil {
		return stored, c.err
	}

	key, assembled, d, err := assembleEntity(c.Entity, c.Ctx, inData)
	if err != nil {
		return stored, err
	}

	stored.Data = append(stored.Data, d.Data...)

	_, err = datastore.Put(c.Ctx, key, assembled)

	return stored, err
}

func Put(ctx context.Context, key *datastore.Key, list datastore.PropertyList) (*datastore.Key, error) {
	return datastore.Put(ctx, key, &list)
}

func (c *Connection) PutMulti(inData []map[string]interface{}) (*Stored, error) {
	var stored = new(Stored)
	var err error

	if c.err != nil {
		return stored, c.err
	}

	var keys []*datastore.Key
	var entities []datastore.PropertyList

	for _, ent := range inData {
		key, assembled, d, err := assembleEntity(c.Entity, c.Ctx, ent)
		if err != nil {
			return stored, err
		}

		keys = append(keys, key)
		entities = append(entities, assembled)
		stored.Data = append(stored.Data, d.Data...)
	}

	_, err = datastore.PutMulti(c.Ctx, keys, entities)

	return stored, err
}
