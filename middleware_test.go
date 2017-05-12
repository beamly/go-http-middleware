package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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
		})
	}
}
