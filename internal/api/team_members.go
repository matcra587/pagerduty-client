package api

import (
	"context"
)

// TeamMember represents a member of a PagerDuty team.
type TeamMember struct {
	User struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	} `json:"user"`
}

// ListTeamMembers returns all members of the given team, auto-paginating.
func (c *Client) ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	var members []TeamMember
	err := paginate(ctx, c, paginateRequest{
		path: "/teams/" + teamID + "/members",
		key:  "members",
	}, func(page []TeamMember) {
		members = append(members, page...)
	})
	if err != nil {
		return nil, err
	}
	return members, nil
}
