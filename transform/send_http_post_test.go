package transform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brexhq/substation/v2/config"
	"github.com/brexhq/substation/v2/message"
)

var _ Transformer = &sendHTTPPost{}

// Statuses that the HTTP client does not retry are returned with a nil error,
// so the transform must inspect the status code itself. Retried statuses (429
// and 5xx) are deliberately excluded here because exercising them would incur
// the client's full backoff and exceed the test timeout.
var sendHTTPPostTests = []struct {
	name       string
	statusCode int
	body       string
	expectErr  bool
}{
	{"200 succeeds", http.StatusOK, "", false},
	{"201 succeeds", http.StatusCreated, "", false},
	{"400 errors", http.StatusBadRequest, "invalid JSON at offset 12", true},
	{"401 errors", http.StatusUnauthorized, "invalid ingest token", true},
	{"404 errors", http.StatusNotFound, "no such endpoint", true},
}

func TestSendHTTPPost(t *testing.T) {
	ctx := context.TODO()

	for _, test := range sendHTTPPostTests {
		t.Run(test.name, func(t *testing.T) {
			serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.statusCode)
				//nolint:errcheck // Test server write.
				w.Write([]byte(test.body))
			}))
			defer serv.Close()

			tf, err := newSendHTTPPost(ctx, config.Config{
				Settings: map[string]interface{}{"url": serv.URL},
			})
			if err != nil {
				t.Fatal(err)
			}

			if _, err := tf.Transform(ctx, message.New().SetData([]byte(`{"a":1}`))); err != nil {
				t.Fatal(err)
			}

			// The batch is sent when the control message is received.
			_, err = tf.Transform(ctx, message.New().AsControl())

			if !test.expectErr {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("expected an error for status %d, got nil", test.statusCode)
			}

			if !strings.Contains(err.Error(), test.body) {
				t.Errorf("expected error to contain response body %q, got %q", test.body, err)
			}
		})
	}
}
