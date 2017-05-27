package middleware

import (
	"encoding/json"
	"expvar"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

var (
	lock = sync.RWMutex{}
)

// Middleware handles and stores state for the middleware
// it's self. It, by and large, wraps our handlers and loggers
type Middleware struct {
	handler http.Handler
	logger  *log.Logger

	// Requests contains a hit counter for each route, minus sensitive data like passwords
	// it is exported for use in telemetry and monitoring endpoints.
	Requests map[string]*expvar.Int
}

type logEntry struct {
	Duration  string    `json:"duration"`
	IPAddress string    `json:"ip_address"`
	RequestID string    `json:"request_id"`
	Status    int       `json:"status"`
	Time      time.Time `json:"time"`
	URL       string    `json:"url"`
}

// NewMiddleware takes an http handler
// to wrap and returns mutable Middleware object
func NewMiddleware(h http.Handler) *Middleware {
	return &Middleware{
		handler: h,
		logger:  log.New(os.Stdout, "", 0),

		Requests: make(map[string]*expvar.Int),
	}
}

// ServeHTTP wraps our requests and produces useful log lines.
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
	requestID := uuid.NewV4().String()

	t0 := time.Now()

	if strings.HasSuffix(r.URL.String(), "/__/counters") {
		rData := make(map[string]int64)
		for k, v := range m.Requests {
			rData[k] = v.Value()
		}

		resp, _ = json.Marshal(rData)
	} else {
		m.handler.ServeHTTP(rec, r)

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

	duration := time.Now().Sub(t0).String()

	// Log request
	l := logEntry{
		URL:       r.URL.String(),
		Duration:  duration,
		IPAddress: r.RemoteAddr,
		RequestID: requestID,
		Status:    rec.Code,
		Time:      t0,
	}
	lOut, err := json.Marshal(l)

	if err == nil {
		m.logger.Print(string(lOut))
	} else {
		m.logger.Printf("error marshaling log data: %q", err)
	}

	// Counters
	url := r.URL.String()
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
		m.Requests[url] = expvar.NewInt(uuid.NewV4().String())
		lock.Unlock()
	}

	lock.Lock()
	m.Requests[url].Add(1)
	lock.Unlock()
}
