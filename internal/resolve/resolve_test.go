package resolve

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_IDPassthrough(t *testing.T) {
	t.Parallel()

	r := New(nil)

	id, matches, err := r.resolve(context.Background(), "service", "PSVC001", nil)
	require.NoError(t, err)
	assert.Equal(t, "PSVC001", id)
	assert.Nil(t, matches)
}

func TestResolve_IDPatternVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		isID  bool
	}{
		{"PSVC001", true},
		{"P123456", true},
		{"PTEAM01", true},
		{"Q1ABC2DEF3GH45", true},
		{"PT4KHLK", true},
		{"database", false},
		{"slack", false},
		{"p12345", false},
		{"Alluvial Oracle", false},
		{"DB", false},
		{"API", false},
		{"SRE", false},
		{"P", false},
		{"a", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			r := New(nil)

			id, _, err := r.resolve(context.Background(), "test", tt.input, func(context.Context) ([]Match, error) {
				if tt.isID {
					t.Fatal("fetch should not be called for IDs")
				}

				return nil, nil
			})
			if tt.isID {
				require.NoError(t, err)
				assert.Equal(t, tt.input, id)
			}
		})
	}
}

func TestResolve_SingleMatch(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fetch := func(context.Context) ([]Match, error) {
		return []Match{
			{ID: "PSVC001", Name: "Database Prod"},
			{ID: "PSVC002", Name: "Auth Service"},
			{ID: "PSVC003", Name: "Payment Gateway"},
		}, nil
	}

	id, matches, err := r.resolve(context.Background(), "service", "database", fetch)
	require.NoError(t, err)
	assert.Equal(t, "PSVC001", id)
	assert.Nil(t, matches)
}

func TestResolve_MultipleMatches(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fetch := func(context.Context) ([]Match, error) {
		return []Match{
			{ID: "PSVC001", Name: "Database Prod"},
			{ID: "PSVC002", Name: "Database Staging"},
			{ID: "PSVC003", Name: "Auth Service"},
		}, nil
	}

	id, matches, err := r.resolve(context.Background(), "service", "database", fetch)
	require.NoError(t, err)
	assert.Empty(t, id)
	require.Len(t, matches, 2)
	assert.Equal(t, "PSVC001", matches[0].ID)
	assert.Equal(t, "PSVC002", matches[1].ID)
}

func TestResolve_ZeroMatches(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fetch := func(context.Context) ([]Match, error) {
		return []Match{
			{ID: "PSVC001", Name: "Database Prod"},
		}, nil
	}

	_, _, err := r.resolve(context.Background(), "service", "zzzznothing", fetch)
	require.Error(t, err)
	assert.ErrorContains(t, err, `no service matching "zzzznothing"`)
}

func TestResolve_CachesCandidatesByKind(t *testing.T) {
	t.Parallel()

	r := New(nil)

	var calls int
	fetch := func(context.Context) ([]Match, error) {
		calls++
		return []Match{
			{ID: "PTEAM001", Name: "Database"},
			{ID: "PTEAM002", Name: "Auth"},
		}, nil
	}

	id, matches, err := r.resolve(context.Background(), "team", "database", fetch)
	require.NoError(t, err)
	assert.Equal(t, "PTEAM001", id)
	assert.Nil(t, matches)

	id, matches, err = r.resolve(context.Background(), "team", "auth", fetch)
	require.NoError(t, err)
	assert.Equal(t, "PTEAM002", id)
	assert.Nil(t, matches)
	assert.Equal(t, 1, calls)
}

func TestResolve_DoesNotShareCandidatesAcrossKinds(t *testing.T) {
	t.Parallel()

	r := New(nil)

	teamCalls := 0
	teamFetch := func(context.Context) ([]Match, error) {
		teamCalls++
		return []Match{
			{ID: "PTEAM001", Name: "Database Team"},
		}, nil
	}

	serviceCalls := 0
	serviceFetch := func(context.Context) ([]Match, error) {
		serviceCalls++
		return []Match{
			{ID: "PSVC001", Name: "Database Service"},
		}, nil
	}

	id, matches, err := r.resolve(context.Background(), "team", "database", teamFetch)
	require.NoError(t, err)
	assert.Equal(t, "PTEAM001", id)
	assert.Nil(t, matches)

	id, matches, err = r.resolve(context.Background(), "service", "database", serviceFetch)
	require.NoError(t, err)
	assert.Equal(t, "PSVC001", id)
	assert.Nil(t, matches)

	assert.Equal(t, 1, teamCalls)
	assert.Equal(t, 1, serviceCalls)
}
