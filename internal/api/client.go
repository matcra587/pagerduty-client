// Package api implements the PagerDuty REST API v2 client.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/version"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

var validSegment = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func validatePathSegment(s string) error {
	if s == "" || !validSegment.MatchString(s) {
		return fmt.Errorf("invalid path segment: %q", s)
	}
	return nil
}

func extractPathSegments(path string) []string {
	var segs []string
	for s := range strings.SplitSeq(path, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	return segs
}

func validatePath(path string) error {
	if strings.ContainsRune(path, '?') {
		return fmt.Errorf("path must not contain query string: %q", path)
	}
	for _, seg := range extractPathSegments(path) {
		if err := validatePathSegment(seg); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultBaseURL = "https://api.pagerduty.com"
	defaultTimeout = 30 * time.Second
	acceptHeader   = "application/vnd.pagerduty+json;version=2"
	// PD rate limit is 960 req/min (16/s); 15/s provides a safety margin.
	pdRateLimit     = 15
	maxRetries      = 3
	maxResponseSize = 10 << 20
	maxRetryAfter   = 60 * time.Second
)

// Client is an HTTP client for the PagerDuty REST API v2.
type Client struct {
	baseURL      string
	token        string
	httpClient   *http.Client
	limiter      *rate.Limiter
	sfGroup      singleflight.Group
	userAgent    string
	extraHeaders map[string]string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default PagerDuty API base URL. Plain HTTP
// is rejected unless the host is localhost or 127.0.0.1 (for testing
// against local mocks).
func WithBaseURL(u string) Option {
	return func(c *Client) {
		parsed, err := url.Parse(u)
		if err != nil {
			clog.Warn().Str("url", u).Msg("rejecting malformed base URL")
			return
		}
		host := parsed.Hostname()
		switch {
		case parsed.Scheme == "https":
			c.baseURL = u
		case parsed.Scheme == "http" && (host == "localhost" || host == "127.0.0.1"):
			c.baseURL = u
		default:
			clog.Warn().Str("url", u).Msg("rejecting non-HTTPS base URL")
		}
	}
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithExtraHeaders adds headers to every request. Headers set here
// override the client defaults (Accept, Content-Type, etc.).
func WithExtraHeaders(headers map[string]string) Option {
	return func(c *Client) {
		c.extraHeaders = headers
	}
}

// NewClient returns a PagerDuty API client authenticated with the given token.
func NewClient(token string, opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		limiter:   rate.NewLimiter(pdRateLimit, pdRateLimit),
		userAgent: "pagerduty-client/" + version.Version,
	}
	for _, o := range opts {
		o(c)
	}
	// Enforce redirect policy after options so WithHTTPClient cannot
	// accidentally undo the hardening.
	c.httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return c
}

type pdErrorBody struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (c *Client) do(ctx context.Context, req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Token token="+c.token)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}

	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		_ = req.Body.Close()
	}

	backoff := time.Second

	for attempt := range maxRetries + 1 {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		resp, err := c.httpClient.Do(req) //nolint:gosec // URL constructed from validated base + path
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, err)
			}
			backoff *= 2
			if sleepErr := sleepContext(ctx, rand.N(max(backoff, 1))); sleepErr != nil { //nolint:gosec // jitter does not need crypto rand
				return nil, sleepErr
			}
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
		if readErr == nil && len(body) > maxResponseSize {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("response body too large (>%d bytes)", maxResponseSize)
		}
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading response body: %w", readErr)
		}

		switch {
		case resp.StatusCode >= 300 && resp.StatusCode < 400:
			return nil, &APIError{StatusCode: resp.StatusCode, Message: "unexpected redirect"}

		case resp.StatusCode == http.StatusTooManyRequests:
			if attempt == maxRetries {
				return nil, &APIError{StatusCode: resp.StatusCode, Message: "rate limited"}
			}
			backoff *= 2
			wait := retryAfterDuration(resp.Header.Get("Retry-After"), rand.N(max(backoff, 1))) //nolint:gosec // jitter does not need crypto rand
			if sleepErr := sleepContext(ctx, wait); sleepErr != nil {
				return nil, sleepErr
			}
			continue

		case resp.StatusCode >= 500:
			if attempt == maxRetries {
				return nil, &APIError{StatusCode: resp.StatusCode, Message: "server error"}
			}
			backoff *= 2
			if sleepErr := sleepContext(ctx, rand.N(max(backoff, 1))); sleepErr != nil { //nolint:gosec // jitter does not need crypto rand
				return nil, sleepErr
			}
			continue

		case resp.StatusCode >= 400:
			var errBody pdErrorBody
			if jsonErr := json.Unmarshal(body, &errBody); jsonErr == nil && errBody.Error.Message != "" {
				return nil, &APIError{
					StatusCode: resp.StatusCode,
					Code:       errBody.Error.Code,
					Message:    errBody.Error.Message,
				}
			}
			return nil, &APIError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}

		default:
			return body, nil
		}
	}

	return nil, errors.New("unexpected end of retry loop")
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	// The first caller's context governs the shared request. This is safe
	// because all callers in this codebase share the same root context.
	v, err, _ := c.sfGroup.Do(u, func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}
		return c.do(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

func (c *Client) postFrom(ctx context.Context, path string, payload any, from string) ([]byte, error) {
	if from == "" {
		return nil, errors.New("from email is required for write operations")
	}
	if err := validatePath(path); err != nil {
		return nil, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("From", from)
	return c.do(ctx, req)
}

func (c *Client) putFrom(ctx context.Context, path string, payload any, from string) ([]byte, error) {
	if from == "" {
		return nil, errors.New("from email is required for write operations")
	}
	if err := validatePath(path); err != nil {
		return nil, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("From", from)
	return c.do(ctx, req)
}

func retryAfterDuration(header string, fallback time.Duration) time.Duration {
	if header == "" {
		return fallback
	}
	secs, err := strconv.Atoi(header)
	if err != nil || secs < 0 {
		return fallback
	}
	return min(max(time.Duration(secs)*time.Second, time.Second), maxRetryAfter)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
