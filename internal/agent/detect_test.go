package agent_test

import (
	"testing"

	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearAgentEnv(t *testing.T) {
	t.Helper()

	vars := []string{
		"CLAUDE_CODE", "CLAUDECODE", "CURSOR_AGENT", "CODEX", "OPENAI_CODEX",
		"AIDER", "CLINE", "WINDSURF_AGENT", "GITHUB_COPILOT",
		"AMAZON_Q", "AWS_Q_DEVELOPER", "GEMINI_CODE_ASSIST", "SRC_CODY", "FORCE_AGENT_MODE",
	}

	for _, v := range vars {
		t.Setenv(v, "")
	}
}

func TestDetect_NoAgent(t *testing.T) {
	clearAgentEnv(t)

	r := agent.Detect()
	assert.False(t, r.Active)
	assert.Empty(t, r.Name)
}

func TestDetect_ClaudeCode(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "1")

	r := agent.Detect()
	require.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetect_ClaudeCodeAlt(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDECODE", "true")

	r := agent.Detect()
	require.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetect_Cursor(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CURSOR_AGENT", "1")

	r := agent.Detect()
	require.True(t, r.Active)
	assert.Equal(t, "Cursor", r.Name)
}

func TestDetect_Codex(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CODEX", "1")

	r := agent.Detect()
	require.True(t, r.Active)
	assert.Equal(t, "Codex", r.Name)
}

func TestDetect_FalseValue(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "0")

	r := agent.Detect()
	assert.False(t, r.Active)
}

func TestDetect_FalseString(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "false")

	r := agent.Detect()
	assert.False(t, r.Active)
}

func TestDetect_NoString(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "no")

	r := agent.Detect()
	assert.False(t, r.Active)
}

func TestDetect_ForceAgentMode(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("FORCE_AGENT_MODE", "1")

	r := agent.Detect()
	require.True(t, r.Active)
	assert.Equal(t, "Unknown", r.Name)
}

func TestDetectWithFlag_FlagTrue_NoEnv(t *testing.T) {
	clearAgentEnv(t)

	r := agent.DetectWithFlag(true)
	assert.True(t, r.Active)
}

func TestDetectWithFlag_FlagTrue_WithEnv(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE", "1")

	r := agent.DetectWithFlag(true)
	require.True(t, r.Active)
	assert.Equal(t, "Claude Code", r.Name)
}

func TestDetectWithFlag_FlagFalse_NoEnv(t *testing.T) {
	clearAgentEnv(t)

	r := agent.DetectWithFlag(false)
	assert.False(t, r.Active)
}

func TestDetectWithFlag_FlagFalse_WithEnv(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CURSOR_AGENT", "1")

	r := agent.DetectWithFlag(false)
	require.True(t, r.Active)
	assert.Equal(t, "Cursor", r.Name)
}
