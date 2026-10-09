package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWorkoutTransportRedactsFailuresAndRejectsRedirects(t *testing.T) {
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer target.Close()
	for _, status := range []int{302, 404, 409, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == 302 {
					w.Header().Set("Location", target.URL)
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte("sensitive-server-body"))
			}))
			defer server.Close()
			var out any
			err := NewClient(server.URL).FetchWorkoutJSON(context.Background(), "/workout-details", url.Values{"uid": {"private-uid"}}, &out)
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
				t.Fatal(err)
			}
			if strings.Contains(err.Error(), "private-uid") || strings.Contains(err.Error(), "sensitive-server-body") || leaked {
				t.Fatal("leaked data or followed redirect")
			}
		})
	}
}
func TestWorkoutTransportBoundsAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", (2<<20)+1))) }))
	defer server.Close()
	client := NewClient(server.URL)
	var out any
	if err := client.FetchWorkoutJSON(context.Background(), "/workout-details", nil, &out); err == nil {
		t.Fatal("accepted oversized response")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.FetchWorkoutJSON(ctx, "/workout-details", nil, &out); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
