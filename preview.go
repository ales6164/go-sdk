package sdk

import "net/http"

// say hello
// todo: rendering engine
func (a *SDK) Render(domain string, dir string) {
	fs := http.FileServer(http.Dir(dir))
	//http.Handle("/preview/", http.StripPrefix("/preview", fs))
	http.Handle("/", http.StripPrefix("/", fs))
}
