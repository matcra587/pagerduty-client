package config_test

import (
	"errors"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTeamID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "valid short", in: "PAB", want: true},
		{name: "valid long", in: "PABCDEF123", want: true},
		{name: "too short", in: "P", want: false},
		{name: "empty", in: "", want: false},
		{name: "wrong prefix", in: "XABC", want: false},
		{name: "lowercase chars", in: "Pabc", want: false},
		{name: "mixed case", in: "PABc", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, config.IsTeamID(tt.in))
		})
	}
}

func TestResolveTeam_ExactID(t *testing.T) {
	listFn := func(_ string) ([]config.Team, error) {
		t.Fatal("listFn should not be called for an ID")
		return nil, nil
	}
	id, err := config.ResolveTeam("PABC123", listFn)
	require.NoError(t, err)
	assert.Equal(t, "PABC123", id)
}

func TestResolveTeam_SingleMatch(t *testing.T) {
	listFn := func(_ string) ([]config.Team, error) {
		return []config.Team{{ID: "PTEAM01", Name: "Platform"}}, nil
	}
	id, err := config.ResolveTeam("Platform", listFn)
	require.NoError(t, err)
	assert.Equal(t, "PTEAM01", id)
}

func TestResolveTeam_NoMatch(t *testing.T) {
	listFn := func(_ string) ([]config.Team, error) {
		return nil, nil
	}
	_, err := config.ResolveTeam("Ghost", listFn)
	require.Error(t, err)
	assert.ErrorContains(t, err, "no team found matching")
}

func TestResolveTeam_Ambiguous(t *testing.T) {
	listFn := func(_ string) ([]config.Team, error) {
		return []config.Team{
			{ID: "PTEAM01", Name: "Platform"},
			{ID: "PTEAM02", Name: "Platform-2"},
		}, nil
	}
	_, err := config.ResolveTeam("Platform", listFn)
	require.Error(t, err)
	require.ErrorContains(t, err, "ambiguous team name")
	require.ErrorContains(t, err, "PTEAM01")
	assert.ErrorContains(t, err, "PTEAM02")
}

func TestResolveTeam_ListError(t *testing.T) {
	listFn := func(_ string) ([]config.Team, error) {
		return nil, errors.New("api timeout")
	}
	_, err := config.ResolveTeam("Platform", listFn)
	require.Error(t, err)
	assert.ErrorContains(t, err, "api timeout")
}
