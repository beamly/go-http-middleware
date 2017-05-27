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
          m := middleware.NewMiddleware(API{})
          http.Handle("/", m)
          panic(http.ListenAndServe(":8008", nil))
  }

  type API struct{}

  func (a API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
          fmt.Fprintf(w, "Hello, %q", r.URL.Path)
  }

A request to `localhost:8008` would then log out:
  {"duration":"34.679µs","ip_address":"[::1]:62865","request_id":"1a3d633b-69c3-4131-9f5b-93274e8c39ae","status":200,"time":"2017-05-27T15:23:33.437735653+01:00","url":"/"}

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
  < Content-Length: 10
  Content-Length: 10

  <
  * Curl_http_done: called premature == 0
  * Connection #0 to host localhost left intact

**/
package middleware
