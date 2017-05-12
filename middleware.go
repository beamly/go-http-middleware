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

// NewMiddleware takes an http handler and an 'app' name
// to wrap and returns mutable Middleware object
func NewMiddleware(h http.Handler, app string) *Middleware {
	return &Middleware{
		handler: h,
		logger:  log.New(os.Stdout, app, log.LstdFlags|log.LUTC|log.Lshortfile),
	}
}

// ServeHTTP wraps our requests and produces useful log lines.
// This happens by intercepting the response which the default handler
// responds with and then sending that on outselves. This approach adds
// latency to a response, but it gives us access to things like status codes -
// information which we absolutely need.
//
// Log lines are produced as per:
//   sample-app2017/05/12 11:34:44 middleware.go:67: {"duration":"32.995µs","ip_address":"[::1]:63841","request_id":"dddb1267-166d-46c0-94d4-3f4f2ceed1f7","status":"200","url":"/"}
// where `sample-app' is the 'app' string passed into NewMiddleware()
//
// These logs are written to STDOUT
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
