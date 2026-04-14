package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

func BenchmarkPaginateServices(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("offset") {
		case "", "0":
			_, _ = w.Write([]byte(`{"services":[{"id":"PSVC1","name":"Web App"},{"id":"PSVC2","name":"Payments"}],"limit":2,"offset":0,"more":true,"total":4}`))
		case "2":
			_, _ = w.Write([]byte(`{"services":[{"id":"PSVC3","name":"Mobile App"},{"id":"PSVC4","name":"API"}],"limit":2,"offset":2,"more":false,"total":4}`))
		default:
			panic("unexpected offset: " + r.URL.Query().Get("offset"))
		}
	})

	server := httptest.NewServer(mux)
	b.Cleanup(server.Close)

	client := NewClient("test-token", WithBaseURL(server.URL))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		var all []pagerduty.Service
		err := paginate(context.Background(), client, paginateRequest{
			path: "/services",
			key:  "services",
		}, func(page []pagerduty.Service) {
			all = append(all, page...)
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(all) != 4 {
			b.Fatalf("expected 4 services, got %d", len(all))
		}
	}
}
