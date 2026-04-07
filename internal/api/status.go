package api

import (
	"context"
	"net/http"
	"time"
)

const probeTimeout = 5 * time.Second

// ProbeAPI checks whether the PagerDuty API at baseURL is reachable.
// It sends an unauthenticated GET to /abilities and returns true if
// any HTTP response is received (including 401). Returns false on
// connection errors, timeouts or 5xx responses. The HTTP status code
// is returned for diagnostic display (0 when unreachable).
func ProbeAPI(ctx context.Context, baseURL string) (ok bool, statusCode int) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/abilities", nil)
	if err != nil {
		return false, 0
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return false, 0
	}
	_ = resp.Body.Close()

	return resp.StatusCode < 500, resp.StatusCode
}
