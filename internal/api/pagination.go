package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
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

type pageEnvelope struct {
	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
	More   bool `json:"more"`
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

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		var envelope pageEnvelope
		if v, ok := raw["limit"]; ok {
			if err := json.Unmarshal(v, &envelope.Limit); err != nil {
				return fmt.Errorf("decoding pagination limit: %w", err)
			}
		}
		if v, ok := raw["offset"]; ok {
			if err := json.Unmarshal(v, &envelope.Offset); err != nil {
				return fmt.Errorf("decoding pagination offset: %w", err)
			}
		}
		if v, ok := raw["more"]; ok {
			if err := json.Unmarshal(v, &envelope.More); err != nil {
				return fmt.Errorf("decoding pagination more: %w", err)
			}
		}

		if envelope.Limit == 0 {
			return errors.New("invalid pagination response: limit is 0")
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

		offset = envelope.Offset + envelope.Limit
		if offset >= pdOffsetCap {
			break
		}
	}

	return nil
}
