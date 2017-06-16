package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const (
	TestResponseBody = "hello, world!"
)

var (
	TestURL, _ = url.Parse("https://user:pass@example.com")
)

type TestAPI struct{}

func (ta TestAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, TestResponseBody)
}

type TestFourOhFourAPI struct{}

func (ta TestFourOhFourAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	fmt.Fprint(w, TestResponseBody)
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
		title          string
		handler        http.Handler
		r              *http.Request
		expectedStatus float64
	}{
		{"a request with user details", TestAPI{}, &http.Request{URL: TestURL}, 200},
		{"a request with user details that 404s", TestFourOhFourAPI{}, &http.Request{URL: TestURL}, 404},
	} {
		t.Run(test.title, func(t *testing.T) {
			m := NewMiddleware(test.handler)

			logWriter := &TestWriter{}
			m.loggers[0].(defaultLogger).output.SetOutput(logWriter)

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
				// Because we go func out for logging and metrics and so on, there is
				// a very real condition where our request returns before logs are written.
				//
				// This is super important for speeding up responses, but is a bit rubbish
				// for testing log output. Thus: we cripple our tests (which is infinitely
				// preferable to crippling response times) by having a little sleep here.
				// We could improve this by waiting and timing out, of course.
				time.Sleep(500 * time.Millisecond)

				t.Run("has data", func(t *testing.T) {
					if len(logWriter.body) == 0 {
						t.Errorf("Nothing was logged within 500ms of response")
					}
				})

				var raw interface{}
				_ = json.Unmarshal(logWriter.body, &raw)

				t.Run("removes password", func(t *testing.T) {
					if raw.(map[string]interface{})["url"].(string) != "https://user@example.com" {
						t.Errorf("expected 'https://user@example.com', received %q", raw.(map[string]interface{})["url"].(string))
					}
				})

				t.Run("correctly sets status", func(t *testing.T) {
					if raw.(map[string]interface{})["status"].(float64) != test.expectedStatus {
						t.Errorf("expected %v, received %v", test.expectedStatus, raw.(map[string]interface{})["status"].(float64))
					}
				})
			})

			t.Run("counters", func(t *testing.T) {
				v, ok := m.Requests["https://user@example.com"]

				t.Run("creates a request counter", func(t *testing.T) {
					if !ok {
						t.Errorf("No request counter was created with URL %q", TestURL)
					}
				})

				t.Run("has a request counter containing exactly one hit", func(t *testing.T) {
					if v.Value() != 1 {
						t.Errorf("expected '1', received '%d'", v.Value)
					}
				})

			})
		})
	}
}
