package main

import (
	"fmt"
	"net/http"

	"github.com/zeebox/go-http-middleware"
)

func main() {
	m := middleware.NewMiddleware(API{})

	fileLogger, err := NewFileLogger("./access.log")
	if err != nil {
		panic(err)
	}

	m.AddLogger(fileLogger)

	http.Handle("/", m)
	panic(http.ListenAndServe(":8008", nil))
}

type API struct{}

func (a API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", r.URL.Path)
}
