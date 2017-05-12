package middleware

import (
	_ "encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const (
	TestResponseBody = "hello, world!"
	TestAppName      = "middleware-test"
)

var (
	TestURL, _ = url.Parse("https://user:pass@example.com")
)

type TestAPI struct{}

func (ta TestAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, TestResponseBody)

	w.WriteHeader(200)
}

type TestWriter struct {
	body []byte
}

func (w *TestWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	err = nil

	for _, b := range p {
		w.body = append(w.body, b)
	}

	return
}

func TestServeHTTP(t *testing.T) {
	for _, test := range []struct {
		title   string
		handler http.Handler
		r       *http.Request
	}{
		{"a request", TestAPI{}, &http.Request{URL: &url.URL{}}},
		{"a request with user details", TestAPI{}, &http.Request{URL: TestURL}},
	} {
		t.Run(test.title, func(t *testing.T) {
			m := NewMiddleware(test.handler, TestAppName)

			logWriter := &TestWriter{}
			m.logger.SetOutput(logWriter)

			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, test.r)

			t.Run("response", func(t *testing.T) {
				t.Run("has a request ID", func(t *testing.T) {
					v := rec.Header().Get("X-Request-ID")
					if v == "" {
						t.Errorf("Expected request ID")
					}
				})
			})

			t.Run("log", func(t *testing.T) {
				t.Run("has data", func(t *testing.T) {
					if len(logWriter.body) == 0 {
						t.Errorf("Nothing was logged")
					}
				})
			})
		})
	}
}
