

# middleware
`import "github.com/beamly/go-http-middleware"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>
*
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

*




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [type FasthttpHandler](#FasthttpHandler)
* [type LogEntry](#LogEntry)
* [type Loggable](#Loggable)
* [type Middleware](#Middleware)
  * [func New(h interface{}) *Middleware](#New)
  * [func NewMiddleware(h interface{}) (m *Middleware)](#NewMiddleware)
  * [func (m *Middleware) AddLogger(l Loggable)](#Middleware.AddLogger)
  * [func (m *Middleware) ServeFastHTTP(ctx *fasthttp.RequestCtx)](#Middleware.ServeFastHTTP)
  * [func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request)](#Middleware.ServeHTTP)


#### <a name="pkg-files">Package files</a>
[default_logger.go](/src/github.com/beamly/go-http-middleware/default_logger.go) [doc.go](/src/github.com/beamly/go-http-middleware/doc.go) [middleware.go](/src/github.com/beamly/go-http-middleware/middleware.go) 


## <a name="pkg-constants">Constants</a>
``` go
const (
    // This is a v5 UUID generated against the DNS namespace
    // The prescence of this UUID in logs means that UUID generation is broken-
    // specifically that https://golang.org/pkg/crypto/rand/#Read is returning
    // an error.
    //
    // The URL used to seed this UUID is james-is-great.beamly.com. This domain
    // does not exist and is, thus, safe to use.
    DefaultBrokenUUID = "cd9bbcae-e076-549f-82bf-a08e8c838dd3"
)
```




## <a name="FasthttpHandler">type</a> [FasthttpHandler](/src/target/middleware.go?s=847:911#L37)
``` go
type FasthttpHandler interface {
    Handle(*fasthttp.RequestCtx)
}
```
FasthttpHandler represents an opinionated fasthttp
based API handler. Because fasthttp doesn't have the
concept of a handler interface, like net/http does, we
need to build our own.










## <a name="LogEntry">type</a> [LogEntry](/src/target/middleware.go?s=1442:1742#L58)
``` go
type LogEntry struct {
    Duration   string    `json:"duration"`
    DurationMS float64   `json:"duration_ms"`
    IPAddress  string    `json:"ip_address"`
    RequestID  string    `json:"request_id"`
    Status     int       `json:"status"`
    Time       time.Time `json:"time"`
    URL        string    `json:"url"`
}

```
LogEntry holds a particular requests data, metadata










## <a name="Loggable">type</a> [Loggable](/src/target/middleware.go?s=1343:1385#L53)
``` go
type Loggable interface {
    Log(LogEntry)
}
```
Loggable is an interface designed to.... log out










## <a name="Middleware">type</a> [Middleware](/src/target/middleware.go?s=1034:1289#L43)
``` go
type Middleware struct {

    // Requests contains a hit counter for each route, minus sensitive data like passwords
    // it is exported for use in telemetry and monitoring endpoints.
    Requests map[string]*expvar.Int
    // contains filtered or unexported fields
}

```
Middleware handles and stores state for the middleware
it's self. It, by and large, wraps our handlers and loggers







### <a name="New">func</a> [New](/src/target/middleware.go?s=2397:2432#L93)
``` go
func New(h interface{}) *Middleware
```
New wraps NewMiddleware- it exists to remove the stutter from
middleware.NewMiddleware and provide a nicer developer experience


### <a name="NewMiddleware">func</a> [NewMiddleware](/src/target/middleware.go?s=1897:1946#L72)
``` go
func NewMiddleware(h interface{}) (m *Middleware)
```
NewMiddleware takes either:


	* a net/http http.Handler; or
	* a middleware.FasthttpHandler

to wrap and returns mutable Middleware object





### <a name="Middleware.AddLogger">func</a> (\*Middleware) [AddLogger](/src/target/middleware.go?s=2615:2657#L100)
``` go
func (m *Middleware) AddLogger(l Loggable)
```
AddLogger takes anything which implements the Loggable interface
and appends it to the Middleware log list which is then used
to log stuff out




### <a name="Middleware.ServeFastHTTP">func</a> (\*Middleware) [ServeFastHTTP](/src/target/middleware.go?s=4709:4769#L163)
``` go
func (m *Middleware) ServeFastHTTP(ctx *fasthttp.RequestCtx)
```
ServeFastHTTP wraps our fasthttp requests and produces useful log lines.
It does this by wrapping fasthttp compliant endopoints and calling them
while timing and catching errors.

Log lines are produced as per:


	{"duration":"394.823µs","ip_address":"[::1]:62405","request_id":"80d1b249-0b43-4adc-9456-e42e0b942ec0","status":200,"time":"2017-05-27T14:57:48.750350842+01:00","url":"/"}

where `sample-app` is the 'app' string passed into NewMiddleware()

These logs are written to `STDOUT`




### <a name="Middleware.ServeHTTP">func</a> (\*Middleware) [ServeHTTP](/src/target/middleware.go?s=3358:3428#L115)
``` go
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
ServeHTTP wraps our net/http requests and produces useful log lines.
This happens by intercepting the response which the default handler
responds with and then sending that on outselves. This approach adds
latency to a response, but it gives us access to things like status codes -
information which we absolutely need.

Log lines are produced as per:


	{"duration":"394.823µs","ip_address":"[::1]:62405","request_id":"80d1b249-0b43-4adc-9456-e42e0b942ec0","status":200,"time":"2017-05-27T14:57:48.750350842+01:00","url":"/"}

where `sample-app` is the 'app' string passed into NewMiddleware()

These logs are written to `STDOUT`








- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
