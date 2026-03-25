package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess_Structure(t *testing.T) {
	data := []string{"item1", "item2"}
	meta := agent.Metadata{Total: 2, More: false, Offset: 0, Limit: 25}
	hints := []string{"use --format json for full output"}

	env := agent.Success("incident list", data, &meta, hints)

	assert.True(t, env.OK)
	assert.Equal(t, "incident list", env.Command)
	assert.Equal(t, data, env.Data)
	require.NotNil(t, env.Meta)
	assert.Equal(t, 2, env.Meta.Total)
	assert.Equal(t, hints, env.Hints)
	assert.Nil(t, env.Err)
}

func TestSuccess_JSONMarshal(t *testing.T) {
	env := agent.Success("incident list", map[string]string{"id": "P123"}, &agent.Metadata{Total: 1}, nil)

	b, err := json.Marshal(env)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))

	assert.Equal(t, true, out["success"])
	assert.Equal(t, "incident list", out["command"])
	assert.NotNil(t, out["data"])
	assert.NotNil(t, out["metadata"])
	assert.Nil(t, out["error"])
}

func TestError_Structure(t *testing.T) {
	env := agent.Error("incident show", 404, "incident not found", "check the incident ID")

	assert.False(t, env.OK)
	assert.Equal(t, "incident show", env.Command)
	assert.Nil(t, env.Data)
	assert.Nil(t, env.Meta)
	require.NotNil(t, env.Err)
	assert.Equal(t, 404, env.Err.Code)
	assert.Equal(t, "incident not found", env.Err.Message)
	assert.Equal(t, "check the incident ID", env.Err.Suggestion)
}

func TestError_JSONMarshal(t *testing.T) {
	env := agent.Error("incident show", 404, "incident not found", "check the incident ID")

	b, err := json.Marshal(env)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))

	assert.Equal(t, false, out["success"])
	assert.Equal(t, "incident show", out["command"])
	assert.Nil(t, out["data"])
	assert.NotNil(t, out["error"])

	errObj, ok := out["error"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 404, errObj["code"], 0)
	assert.Equal(t, "incident not found", errObj["message"])
	assert.Equal(t, "check the incident ID", errObj["suggestion"])
}

func TestMetadata_NilOmitted(t *testing.T) {
	env := agent.Success("test", nil, nil, nil)

	b, err := json.Marshal(env)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Nil(t, out["metadata"])
}

func TestMetadata_OmitEmpty(t *testing.T) {
	env := agent.Success("test", nil, &agent.Metadata{Total: 5}, nil)

	b, err := json.Marshal(env)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))

	meta, ok := out["metadata"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 5, meta["total"], 0)
	assert.Equal(t, false, meta["more"])
	assert.Nil(t, meta["offset"])
	assert.Nil(t, meta["limit"])
}
