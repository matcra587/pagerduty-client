package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginateAll(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		switch offset {
		case "", "0":
			_, _ = w.Write([]byte(`{"items":[{"id":"1"},{"id":"2"}],"limit":2,"offset":0,"more":true,"total":4}`))
		case "2":
			_, _ = w.Write([]byte(`{"items":[{"id":"3"},{"id":"4"}],"limit":2,"offset":2,"more":false,"total":4}`))
		default:
			t.Fatalf("unexpected offset: %s", offset)
		}
	})

	c := NewClient("test-token", WithBaseURL(server.URL))

	type item struct {
		ID string `json:"id"`
	}

	var all []item
	err := paginate(context.Background(), c, paginateRequest{
		path: "/items",
		key:  "items",
	}, func(items []item) {
		all = append(all, items...)
	})

	require.NoError(t, err)
	assert.Len(t, all, 4)
	assert.Equal(t, "1", all[0].ID)
	assert.Equal(t, "4", all[3].ID)
}

func TestPaginateWithLimit(t *testing.T) {
	requestCount := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write(fmt.Appendf(nil,
			`{"items":[{"id":"%d"}],"limit":1,"offset":%d,"more":true,"total":100}`,
			requestCount, requestCount-1,
		))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))

	type item struct {
		ID string `json:"id"`
	}

	var all []item
	err := paginate(context.Background(), c, paginateRequest{
		path: "/items",
		key:  "items",
	}, func(items []item) {
		all = append(all, items...)
	}, withMaxResults(3))

	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestPaginateStopsAtOffsetCap(t *testing.T) {
	requestCount := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		_, _ = w.Write(fmt.Appendf(nil,
			`{"items":[{"id":"%d"}],"limit":100,"offset":%d,"more":true,"total":20000}`,
			requestCount, offset,
		))
	})

	c := NewClient("test-token", WithBaseURL(server.URL))

	type item struct {
		ID string `json:"id"`
	}

	var all []item
	err := paginate(context.Background(), c, paginateRequest{
		path: "/items",
		key:  "items",
	}, func(items []item) {
		all = append(all, items...)
	})

	require.NoError(t, err)
	assert.Equal(t, 100, requestCount)
	assert.Len(t, all, 100)
}

func TestPaginateCancelledContext(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	page := 0
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		page++
		_, _ = w.Write(fmt.Appendf(nil,
			`{"items":[{"id":"%d"}],"limit":1,"offset":%d,"more":true,"total":100}`,
			page, page-1,
		))
	})

	ctx, cancel := context.WithCancel(context.Background())

	c := NewClient("test-token", WithBaseURL(server.URL))

	type item struct {
		ID string `json:"id"`
	}

	var all []item
	err := paginate(ctx, c, paginateRequest{
		path: "/items",
		key:  "items",
	}, func(items []item) {
		all = append(all, items...)
		if len(all) >= 2 {
			cancel()
		}
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled"),
		"expected context cancellation error, got: %v", err)
}
