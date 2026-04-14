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

type decodedPage[T any] struct {
	Limit       uint
	Offset      uint
	More        bool
	Items       []T
	HasItemsKey bool
}

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

func decodePaginatedPage[T any](body []byte, key string) (decodedPage[T], error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return decodedPage[T]{}, fmt.Errorf("decoding response: %w", err)
	}

	var page decodedPage[T]
	if rawLimit, ok := raw["limit"]; ok {
		if err := json.Unmarshal(rawLimit, &page.Limit); err != nil {
			return page, fmt.Errorf("decoding pagination envelope: %w", err)
		}
	}
	if rawOffset, ok := raw["offset"]; ok {
		if err := json.Unmarshal(rawOffset, &page.Offset); err != nil {
			return page, fmt.Errorf("decoding pagination envelope: %w", err)
		}
	}
	if rawMore, ok := raw["more"]; ok {
		if err := json.Unmarshal(rawMore, &page.More); err != nil {
			return page, fmt.Errorf("decoding pagination envelope: %w", err)
		}
	}

	rawItems, ok := raw[key]
	page.HasItemsKey = ok
	if !ok {
		return page, nil
	}

	if err := json.Unmarshal(rawItems, &page.Items); err != nil {
		return page, fmt.Errorf("decoding %s: %w", key, err)
	}

	return page, nil
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

		page, err := decodePaginatedPage[T](body, req.key)
		if err != nil {
			return err
		}
		if page.Limit == 0 {
			return errors.New("invalid pagination response: limit is 0")
		}
		if !page.HasItemsKey {
			break
		}

		items := page.Items
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

		if !page.More {
			break
		}

		offset = int(page.Offset) + int(page.Limit) //nolint:gosec // PD offset capped at 10,000
		if offset >= pdOffsetCap {
			break
		}
	}

	return nil
}
