package api

import (
	"context"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// LogEntryOpts filters for listing incident log entries.
type LogEntryOpts struct {
	Since      string
	Until      string
	IsOverview bool
}

// ListIncidentLogEntries returns log entries for the given incident, auto-paginating.
func (c *Client) ListIncidentLogEntries(ctx context.Context, incidentID string, opts LogEntryOpts) ([]pagerduty.LogEntry, error) {
	params := logEntryParams(opts)
	var entries []pagerduty.LogEntry
	err := paginate(ctx, c, paginateRequest{
		path:   "/incidents/" + incidentID + "/log_entries",
		params: params,
		key:    "log_entries",
	}, func(page []pagerduty.LogEntry) {
		entries = append(entries, page...)
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func logEntryParams(opts LogEntryOpts) url.Values {
	v := url.Values{}
	if opts.Since != "" {
		v.Set("since", opts.Since)
	}
	if opts.Until != "" {
		v.Set("until", opts.Until)
	}
	if opts.IsOverview {
		v.Set("is_overview", "true")
	}
	return v
}
