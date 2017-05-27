package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"github.com/satori/go.uuid"
)

// Middleware handles and stores state for the middleware
// it's self. It, by and large, wraps our handlers and loggers
type Middleware struct {
	handler http.Handler
	logger  *log.Logger
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
	rec := httptest.NewRecorder()
	requestID := uuid.NewV4().String()

	t0 := time.Now()
	m.handler.ServeHTTP(rec, r)
	duration := time.Now().Sub(t0).String()

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
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(rec.Code)
	w.Write(rec.Body.Bytes())

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
}
