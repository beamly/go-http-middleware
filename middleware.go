package middleware

import (
	"encoding/json"
	"expvar"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"
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

// Middleware handles and stores state for the middleware
// it's self. It, by and large, wraps our handlers and loggers
type Middleware struct {
	handler http.Handler
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
		loggers: []Loggable{newDefaultLogger()},

		Requests: make(map[string]*expvar.Int),
	}
}

// AddLogger takes anything which implements the Loggable interface
// and appends it to the Middleware log list which is then used
// to log stuff out
func (m *Middleware) AddLogger(l Loggable) {
	m.loggers = append(m.loggers, l)
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

	requestID := newUUID()
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

	// Do the rest asynchronously; there's no point blocking threads/ connections
	// further

	go func() {
		duration := time.Now().Sub(t0).String()

		// Log request
		l := LogEntry{
			URL:       r.URL.String(),
			Duration:  duration,
			IPAddress: r.RemoteAddr,
			RequestID: requestID,
			Status:    rec.Code,
			Time:      t0,
		}

		for _, logger := range m.loggers {
			go logger.Log(l)
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
			m.Requests[url] = expvar.NewInt(newUUID())
			lock.Unlock()
		}

		lock.Lock()
		m.Requests[url].Add(1)
		lock.Unlock()
	}()
}

func newUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return DefaultBrokenUUID
	}

	return u.String()
}
