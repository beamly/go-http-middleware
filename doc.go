/**
Package middleware provides a net/http middleware for APIs and web servers.

It largely works in much the same way as `node-autoscaling` does: it wraps a struct that
implements `http.Handler` by:

1. Minting a requestID
2. Running, and timing, the wrapped handler
3. Adding the request ID to the response
4. _responding_ with this data
5. Logging out, to STDOUT, request metadata


A simple implementation would look like:

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

A request to `localhost:8008` would then log out:

  sample-app2017/05/12 11:45:54 middleware.go:86: {"duration":"56.581µs","ip_address":"[::1]:63985","request_id":"abb0969e-0879-4838-bb5f-3c018f34ab17","status":200,"time":"2017-05-12T12:45:54.1477631+01:00","url":"/"}

With the response:

  *   Trying ::1...
  * TCP_NODELAY set
  * Connected to localhost (::1) port 8008 (#0)
  > HEAD / HTTP/1.1
  > Host: localhost:8008
  > User-Agent: curl/7.51.0
  > Accept: text/plain
  >
  < HTTP/1.1 200 OK
  HTTP/1.1 200 OK
  < Content-Type: text/plain; charset=utf-8
  Content-Type: text/plain; charset=utf-8
  < X-Request-Id: 72c2b7aa-3bcb-478f-8724-66f38cd3abc0
  X-Request-Id: 72c2b7aa-3bcb-478f-8724-66f38cd3abc0
  < Date: Fri, 12 May 2017 11:55:13 GMT
  Date: Fri, 12 May 2017 11:55:13 GMT
  < Content-Length: 10
  Content-Length: 10

  <
  * Curl_http_done: called premature == 0
  * Connection #0 to host localhost left intact

**/
package middleware
