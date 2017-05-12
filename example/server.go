package main

import (
	"fmt"
	"net/http"

	"github.com/zeebox/go-http-middleware"
)

func main() {
	m := middleware.NewMiddleware(API{}, "sample-app")
	http.Handle("/", m)
	panic(http.ListenAndServe(":8008", nil))
}

type API struct{}

func (a API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", r.URL.Path)
}
