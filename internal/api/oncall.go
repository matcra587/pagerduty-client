package api

import (
	"context"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListOnCallsOpts specifies optional filters for listing on-call entries.
type ListOnCallsOpts struct {
	UserIDs             []string
	ScheduleIDs         []string
	EscalationPolicyIDs []string
	Since               string
	Until               string
	Earliest            bool
}

// ListOnCalls returns all on-call entries matching opts via paginated GET /oncalls.
func (c *Client) ListOnCalls(ctx context.Context, opts ListOnCallsOpts) ([]pagerduty.OnCall, error) {
	params := onCallParams(opts)

	var results []pagerduty.OnCall
	collect := func(page []pagerduty.OnCall) {
		results = append(results, page...)
	}

	if err := paginate[pagerduty.OnCall](ctx, c, paginateRequest{
		path:   "/oncalls",
		params: params,
		key:    "oncalls",
	}, collect); err != nil {
		return nil, fmt.Errorf("listing on-calls: %w", err)
	}

	return results, nil
}

func onCallParams(opts ListOnCallsOpts) url.Values {
	v := url.Values{}
	for _, id := range opts.UserIDs {
		v.Add("user_ids[]", id)
	}
	for _, id := range opts.ScheduleIDs {
		v.Add("schedule_ids[]", id)
	}
	for _, id := range opts.EscalationPolicyIDs {
		v.Add("escalation_policy_ids[]", id)
	}
	if opts.Since != "" {
		v.Set("since", opts.Since)
	}
	if opts.Until != "" {
		v.Set("until", opts.Until)
	}
	if opts.Earliest {
		v.Set("earliest", "true")
	}
	return v
}
