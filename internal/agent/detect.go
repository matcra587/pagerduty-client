// Package agent provides agent detection and structured envelope formatting.
package agent

import (
	"os"
	"strings"
)

// DetectionResult holds the outcome of agent environment detection.
type DetectionResult struct {
	Active bool
	Name   string
}

var envVars = []struct {
	key  string
	name string
}{
	{"CLAUDE_CODE", "Claude Code"},
	{"CLAUDECODE", "Claude Code"},
	{"CURSOR_AGENT", "Cursor"},
	{"CODEX", "Codex"},
	{"OPENAI_CODEX", "Codex"},
	{"AIDER", "Aider"},
	{"CLINE", "Cline"},
	{"WINDSURF_AGENT", "Windsurf"},
	{"GITHUB_COPILOT", "GitHub Copilot"},
	{"AMAZON_Q", "Amazon Q"},
	{"AWS_Q_DEVELOPER", "Amazon Q"},
	{"GEMINI_CODE_ASSIST", "Gemini Code Assist"},
	{"SRC_CODY", "Cody"},
	{"FORCE_AGENT_MODE", "Unknown"},
}

func isTruthy(val string) bool {
	if val == "" {
		return false
	}

	switch strings.ToLower(val) {
	case "0", "false", "no":
		return false
	}

	return true
}

// Detect checks environment variables for known AI coding agents.
func Detect() DetectionResult {
	for _, e := range envVars {
		if isTruthy(os.Getenv(e.key)) {
			return DetectionResult{Active: true, Name: e.name}
		}
	}

	return DetectionResult{}
}

// DetectWithFlag runs Detect and forces Active to true when flag is set.
// If no agent env var was found, Name is set to "manual".
func DetectWithFlag(flag bool) DetectionResult {
	r := Detect()
	if flag {
		r.Active = true
		if r.Name == "" {
			r.Name = "manual"
		}
	}
	return r
}
