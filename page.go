package sdk

import (
	"path/filepath"
	"net/http"
	"strings"
)

type Page struct {
	Name     string
	path     string
	template *Template
}

func NewPage(path string) *Page {
	t, err := ParseFile(path)
	if err != nil {
		panic(err)
	}

	return &Page{
		path:     path,
		Name:     strings.Split(filepath.Base(path), ".")[0],
		template: t,
	}
}

func (a *SDK) HandlePage(path string, page *Page, context ...interface{}) {
	a.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		page.Render(w, context...)
	})
}

func (a *SDK) HandlePageWithLayout(path string, page *Page, layout *Page, context ...interface{}) {
	a.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		page.RenderInLayout(w, layout, context...)
	})
}

func (p *Page) RenderInLayout(w http.ResponseWriter, layout *Page, context ...interface{}) {
	w.Header().Set("Content-Type", "text/html")
	context = append(context, map[string]string{"page": p.Name})
	buf := p.template.RenderInLayout(layout.template, context...)
	buf.WriteTo(w)
	buf.Reset()

}

func (p *Page) Render(w http.ResponseWriter, context ...interface{}) {
	w.Header().Set("Content-Type", "text/html")
	context = append(context, map[string]string{"page": p.Name})
	buf := p.template.Render(context...)
	buf.WriteTo(w)
	buf.Reset()
}
