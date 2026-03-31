package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEscalationPolicies(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"escalation_policies":[{"id":"P1","name":"Platform","num_loops":2,"escalation_rules":[{"id":"R1","escalation_delay_in_minutes":5,"targets":[{"id":"U1","type":"user_reference","summary":"Alice"}]}]}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	policies, err := client.ListEscalationPolicies(context.Background(), ListEscalationPoliciesOpts{})

	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "P1", policies[0].ID)
	assert.Equal(t, "Platform", policies[0].Name)
	assert.Equal(t, uint(2), policies[0].NumLoops)
	require.Len(t, policies[0].EscalationRules, 1)
	assert.Equal(t, uint(5), policies[0].EscalationRules[0].Delay)
}

func TestListEscalationPolicies_WithQuery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "platform", r.URL.Query().Get("query"))
		_, _ = w.Write([]byte(`{"escalation_policies":[{"id":"P1","name":"Platform"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	policies, err := client.ListEscalationPolicies(context.Background(), ListEscalationPoliciesOpts{Query: "platform"})

	require.NoError(t, err)
	assert.Len(t, policies, 1)
}

func TestListEscalationPolicies_WithTeamFilter(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"T1", "T2"}, r.URL.Query()["team_ids[]"])
		_, _ = w.Write([]byte(`{"escalation_policies":[{"id":"P1","name":"Platform"}],"limit":100,"offset":0,"more":false}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	policies, err := client.ListEscalationPolicies(context.Background(), ListEscalationPoliciesOpts{TeamIDs: []string{"T1", "T2"}})

	require.NoError(t, err)
	assert.Len(t, policies, 1)
}

func TestListEscalationPolicies_Pagination(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var page atomic.Int32
	mux.HandleFunc("GET /escalation_policies", func(w http.ResponseWriter, _ *http.Request) {
		n := page.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{"escalation_policies":[{"id":"P1","name":"Alpha"}],"limit":1,"offset":0,"more":true,"total":2}`))
		} else {
			_, _ = w.Write([]byte(`{"escalation_policies":[{"id":"P2","name":"Bravo"}],"limit":1,"offset":1,"more":false,"total":2}`))
		}
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	policies, err := client.ListEscalationPolicies(context.Background(), ListEscalationPoliciesOpts{})

	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "P1", policies[0].ID)
	assert.Equal(t, "P2", policies[1].ID)
}

func TestGetEscalationPolicy(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /escalation_policies/P1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		assert.Equal(t, []string{"targets"}, r.URL.Query()["include[]"])
		_, _ = w.Write([]byte(`{"escalation_policy":{"id":"P1","name":"Platform","description":"Primary platform EP","num_loops":2,"escalation_rules":[{"id":"R1","escalation_delay_in_minutes":5,"targets":[{"id":"U1","type":"user_reference","summary":"Alice Smith"}]},{"id":"R2","escalation_delay_in_minutes":15,"targets":[{"id":"U2","type":"user_reference","summary":"Bob Jones"},{"id":"S1","type":"schedule_reference","summary":"Primary Schedule"}]}],"teams":[{"id":"T1","type":"team_reference"}]}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	ep, err := client.GetEscalationPolicy(context.Background(), "P1")

	require.NoError(t, err)
	require.NotNil(t, ep)
	assert.Equal(t, "P1", ep.ID)
	assert.Equal(t, "Platform", ep.Name)
	assert.Equal(t, "Primary platform EP", ep.Description)
	assert.Equal(t, uint(2), ep.NumLoops)
	require.Len(t, ep.EscalationRules, 2)
	assert.Equal(t, uint(5), ep.EscalationRules[0].Delay)
	require.Len(t, ep.EscalationRules[0].Targets, 1)
	assert.Equal(t, "Alice Smith", ep.EscalationRules[0].Targets[0].Summary)
	require.Len(t, ep.EscalationRules[1].Targets, 2)
	require.Len(t, ep.Teams, 1)
	assert.Equal(t, "T1", ep.Teams[0].ID)
}

func TestGetEscalationPolicy_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("GET /escalation_policies/NOTFOUND", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Not Found","code":2100}}`))
	})

	client := NewClient("test-token", WithBaseURL(server.URL))
	ep, err := client.GetEscalationPolicy(context.Background(), "NOTFOUND")

	require.Error(t, err)
	assert.Nil(t, ep)
	require.ErrorIs(t, err, ErrNotFound)
}
