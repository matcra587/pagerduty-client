package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/PagerDuty/go-pagerduty"
)

const defaultPageSize = 100

const pdOffsetCap = 10_000

type paginateOption func(*paginateConfig)

type paginateConfig struct {
	maxResults int
}

func withMaxResults(n int) paginateOption {
	return func(cfg *paginateConfig) {
		cfg.maxResults = n
	}
}

type paginateRequest struct {
	path   string
	params url.Values
	key    string
}

func paginate[T any](ctx context.Context, c *Client, req paginateRequest, collect func([]T), opts ...paginateOption) error {
	cfg := &paginateConfig{}
	for _, o := range opts {
		o(cfg)
	}

	pageSize := defaultPageSize
	if cfg.maxResults > 0 && cfg.maxResults < pageSize {
		pageSize = cfg.maxResults
	}

	params := req.params
	if params == nil {
		params = url.Values{}
	}
	merged := make(url.Values, len(params)+2)
	for k, vs := range params {
		merged[k] = append([]string(nil), vs...)
	}

	offset := 0
	collected := 0

	for {
		merged.Set("limit", strconv.Itoa(pageSize))
		merged.Set("offset", strconv.Itoa(offset))

		body, err := c.get(ctx, req.path, merged)
		if err != nil {
			return err
		}

		// Unmarshal envelope fields (limit, offset, more) using the
		// go-pagerduty type. Unknown fields (the data array) are ignored.
		var envelope pagerduty.APIListObject
		if err := json.Unmarshal(body, &envelope); err != nil {
			return fmt.Errorf("decoding pagination envelope: %w", err)
		}

		if envelope.Limit == 0 {
			return errors.New("invalid pagination response: limit is 0")
		}

		// Extract the data array by dynamic key.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		rawItems, ok := raw[req.key]
		if !ok {
			break
		}

		var items []T
		if err := json.Unmarshal(rawItems, &items); err != nil {
			return fmt.Errorf("decoding %s: %w", req.key, err)
		}

		if cfg.maxResults > 0 {
			remaining := cfg.maxResults - collected
			if len(items) > remaining {
				items = items[:remaining]
			}
		}

		if len(items) > 0 {
			collect(items)
			collected += len(items)
		}

		if cfg.maxResults > 0 && collected >= cfg.maxResults {
			break
		}

		if !envelope.More {
			break
		}

		offset = int(envelope.Offset) + int(envelope.Limit) //nolint:gosec // PD offset capped at 10,000
		if offset >= pdOffsetCap {
			break
		}
	}

	return nil
}
