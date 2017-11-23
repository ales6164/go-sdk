package sdk

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/search"
)

type Document struct {
	Fields []search.Field
	Facets []search.Facet
	Value  map[string]interface{}
}

type SearchType struct {

}

func (d *Document) AddFields(f ...search.Field) error {
	d.Fields = append(d.Fields, f...)
	return nil
}

func (d *Document) AddFacets(f ...search.Facet) error {
	d.Facets = append(d.Facets, f...)
	return nil
}

func (d *Document) Load(fields []search.Field, meta *search.DocumentMetadata) error {
	d.Fields = append(d.Fields, fields...)
	d.Facets = append(d.Facets, meta.Facets...)

	d.Value = map[string]interface{}{}
	for _, val := range d.Fields {
		d.Value[val.Name] = val.Value
	}

	return nil
}

func (d *Document) Save() ([]search.Field, *search.DocumentMetadata, error) {
	meta := &search.DocumentMetadata{
		Facets: d.Facets,
	}

	d.Value = map[string]interface{}{}
	for _, val := range d.Fields {
		d.Value[val.Name] = val.Value
	}

	return d.Fields, meta, nil
}

type DocumentDefinition struct {
	Name   string
	Fields []string
	Facets []string
}

func ClearIndex(ctx context.Context, name string) error {
	index, err := search.Open(name)
	if err != nil {
		return err
	}

	var ids []string

	t := index.List(ctx, &search.ListOptions{IDsOnly: true})
	for {
		var emp interface{}
		id, err := t.Next(emp)
		if err == search.Done {
			break // No further entities match the query.
		}
		if err != nil {
			return err
		}
		// Do something with Person p and Key k
		ids = append(ids, id)
	}

	var divided [][]string
	chunkSize := 100
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize

		if end > len(ids) {
			end = len(ids)
		}

		divided = append(divided, ids[i:end])
	}

	for _, chunk := range divided {
		if err = index.DeleteMulti(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (dd *DocumentDefinition) Put(ctx context.Context, id string, data map[string]interface{}) error {
	assembled := dd.Assemble(id, data)

	index, err := search.Open(dd.Name)
	if err != nil {
		return err
	}

	_, err = index.Put(ctx, id, &assembled)
	return err
}

func (dd *DocumentDefinition) Assemble(id string, data map[string]interface{}) Document {
	var document = Document{}

	var f = search.Field{
		Name:  "id",
		Value: id,
	}
	document.AddFields(f)

	for _, name := range dd.Fields {
		var val interface{} = data[name]

		if val != nil {

			if valArr, ok := val.([]interface{}); ok {
				for _, val := range valArr {
					var f = search.Field{
						Name:  name,
						Value: val,
					}
					document.AddFields(f)
				}
			} else {
				var f = search.Field{
					Name:  name,
					Value: val,
				}
				document.AddFields(f)
			}
		}
	}

	for _, name := range dd.Facets {
		var val interface{} = data[name]

		if val != nil {

			if valArr, ok := val.([]interface{}); ok {
				for _, val := range valArr {
					if valStr, ok := val.(string); ok {
						val = search.Atom(valStr)
					}

					var f = search.Facet{
						Name:  name,
						Value: val,
					}
					document.AddFacets(f)
				}
			} else {
				if valStr, ok := val.(string); ok {
					val = search.Atom(valStr)
				}

				var f = search.Facet{
					Name:  name,
					Value: val,
				}
				document.AddFacets(f)
			}
		}
	}

	return document
}
