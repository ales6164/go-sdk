package sdk

import (
	"html/template"
	"net/http"
)

func ParsePage(funcs template.FuncMap, templates ...string) (*template.Template, error) {
	return template.New("").Funcs(htmlFuncMap).Funcs(funcs).ParseFiles(templates...)
}

func RenderTemplate(w http.ResponseWriter, templ *template.Template, data interface{}) error {
	return templ.ExecuteTemplate(w, "index", data)
}

var htmlFuncMap = template.FuncMap{
	"valOfMap": func(x map[string]interface{}, key string) interface{} {
		return x[key]
	},
	"toHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"toCSS": func(s string) template.CSS {
		return template.CSS(s)
	},
	"toJS": func(s string) template.JS {
		return template.JS(s)
	},
}
