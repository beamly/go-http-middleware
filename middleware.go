package middleware

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"
	"github.com/valyala/fasthttp"
)

var (
	lock = sync.RWMutex{}
)

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

// FasthttpHandler represents an opinionated fasthttp
// based API handler. Because fasthttp doesn't have the
// concept of a handler interface, like net/http does, we
// need to build our own.
type FasthttpHandler interface {
	Handle(*fasthttp.RequestCtx)
}

// Middleware handles and stores state for the middleware
// it's self. It, by and large, wraps our handlers and loggers
type Middleware struct {
	handler interface{}
	loggers []Loggable

	// Requests contains a hit counter for each route, minus sensitive data like passwords
	// it is exported for use in telemetry and monitoring endpoints.
	Requests map[string]*expvar.Int
}

// Loggable is an interface designed to.... log out
type Loggable interface {
	Log(LogEntry)
}

// LogEntry holds a particular requests data, metadata
type LogEntry struct {
	Duration   string    `json:"duration"`
	DurationMS float64   `json:"duration_ms"`
	IPAddress  string    `json:"ip_address"`
	RequestID  string    `json:"request_id"`
	Status     int       `json:"status"`
	Time       time.Time `json:"time"`
	URL        string    `json:"url"`
	UserAgent  string    `json:"useragent"`
}

// NewMiddleware takes either:
//    * a net/http http.Handler; or
//    * a middleware.FasthttpHandler
// to wrap and returns mutable Middleware object
func NewMiddleware(h interface{}) (m *Middleware) {
	m = &Middleware{}

	_, isNetHTTP := h.(http.Handler)
	_, isFastHTTP := h.(FasthttpHandler)

	if !isNetHTTP && !isFastHTTP {
		err := fmt.Errorf("Unrecognised interface type %T", h)

		panic(err)
	}

	m.handler = h
	m.loggers = []Loggable{newDefaultLogger()}
	m.Requests = make(map[string]*expvar.Int)

	return
}

// New wraps NewMiddleware- it exists to remove the stutter from
// middleware.NewMiddleware and provide a nicer developer experience
func New(h interface{}) *Middleware {
	return NewMiddleware(h)
}

// AddLogger takes anything which implements the Loggable interface
// and appends it to the Middleware log list which is then used
// to log stuff out
func (m *Middleware) AddLogger(l Loggable) {
	m.loggers = append(m.loggers, l)
}

// ServeHTTP wraps our net/http requests and produces useful log lines.
// This happens by intercepting the response which the default handler
// responds with and then sending that on outselves. This approach adds
// latency to a response, but it gives us access to things like status codes -
// information which we absolutely need.
//
// Log lines are produced as per:
//   {"duration":"394.823µs","ip_address":"[::1]:62405","request_id":"80d1b249-0b43-4adc-9456-e42e0b942ec0","status":200,"time":"2017-05-27T14:57:48.750350842+01:00","url":"/"}
// where `sample-app` is the 'app' string passed into NewMiddleware()
//
// These logs are written to `STDOUT`
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := []byte{}
	status := 200

	rec := httptest.NewRecorder()

	requestID := newUUID()
	t0 := time.Now()

	if strings.HasSuffix(r.URL.String(), "/__/counters") {
		resp = m.counters()
	} else {
		m.handler.(http.Handler).ServeHTTP(rec, r)

		if r.URL.User != nil {
			_, set := r.URL.User.Password()
			if set {
				// ensure passwords aren't leaked
				r.URL.User = url.User(r.URL.User.Username())
			}
		}

		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		resp = rec.Body.Bytes()
		status = rec.Code
	}

	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	w.Write(resp)

	// Do the rest asynchronously; there's no point blocking threads/ connections
	// further

	go m.log(requestID, t0, r.RemoteAddr, rec.Code, r.URL.String(), r.UserAgent())
}

// ServeFastHTTP wraps our fasthttp requests and produces useful log lines.
// It does this by wrapping fasthttp compliant endopoints and calling them
// while timing and catching errors.
//
// Log lines are produced as per:
//   {"duration":"394.823µs","ip_address":"[::1]:62405","request_id":"80d1b249-0b43-4adc-9456-e42e0b942ec0","status":200,"time":"2017-05-27T14:57:48.750350842+01:00","url":"/"}
// where `sample-app` is the 'app' string passed into NewMiddleware()
//
// These logs are written to `STDOUT`
func (m *Middleware) ServeFastHTTP(ctx *fasthttp.RequestCtx) {
	requestID := newUUID()
	ctx.Response.Header.Set("X-Request-ID", requestID)

	if strings.HasSuffix(ctx.URI().String(), "/__/counters") {
		resp := m.counters()

		fmt.Fprintf(ctx, string(resp))
	} else {
		m.handler.(FasthttpHandler).Handle(ctx)
	}

	// Do the rest asynchronously; there's no point blocking threads/ connections
	// further

	go m.log(requestID, ctx.ConnTime(), ctx.RemoteAddr().String(), ctx.Response.StatusCode(), ctx.URI().String(), string(ctx.UserAgent()))
}

func (m *Middleware) counters() (resp []byte) {
	rData := make(map[string]int64)
	for k, v := range m.Requests {
		rData[k] = v.Value()
	}

	resp, _ = json.Marshal(rData)

	return
}

func (m *Middleware) log(requestID string, t0 time.Time, addr string, status int, url string, ua string) {
	duration := time.Now().Sub(t0)

	// Log request
	l := LogEntry{
		Duration:   duration.String(),
		DurationMS: float64(duration / time.Millisecond),
		IPAddress:  addr,
		RequestID:  requestID,
		Status:     status,
		Time:       t0,
		URL:        url,
		UserAgent:  ua,
	}

	for _, logger := range m.loggers {
		go logger.Log(l)
	}

	// Counters
	lock.RLock()
	_, ok := m.Requests[url]
	lock.RUnlock()

	if !ok {
		// On uuids: during development it became obvious that there were possible collisions/ unexpected behaviour
		// around how we store counters.
		// Because we don't know all of the routes exposed, and as such we can't preallocate counters, we store them
		// in a map against route names. This allows us to point to the correct counter. It also means that should multiple
		// *middleware.Middleware instances match the same route (say: an application listening on two ports exposing '/')
		// then by not setting the counter as the route (or similarly computed value) we're not going to end up with both
		// counters being merged into a single one.
		//
		// This was found during testing: initially storing counters named for their route, which expvar makes globally available,
		// in a map, which is stored in an instanced *middleware.Middleware, meant that this function always fired and tried to
		// redfine a counter that existed that `expvar`, in it's wisdom, bombed out on.
		lock.Lock()
		m.Requests[url] = expvar.NewInt(newUUID())
		lock.Unlock()
	}

	lock.Lock()
	m.Requests[url].Add(1)
	lock.Unlock()
}

func newUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return DefaultBrokenUUID
	}

	return u.String()
}
