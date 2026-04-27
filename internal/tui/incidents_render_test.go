package tui

import (
	"strings"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/x/ansi"
	"github.com/matcra587/pagerduty-client/internal/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderBadge(t *testing.T) {
	t.Parallel()
	out := renderBadge(integration.Field{Label: "State", Value: "Triggered"})
	assert.Contains(t, xansi.Strip(out), "Triggered")
}

func TestRenderBadge_CaseInsensitive(t *testing.T) {
	t.Parallel()
	out := renderBadge(integration.Field{Label: "State", Value: "ALARM"})
	assert.Contains(t, xansi.Strip(out), "ALARM")
}

func TestRenderBadge_EmptyValue(t *testing.T) {
	t.Parallel()
	assert.Empty(t, renderBadge(integration.Field{Label: "State", Value: ""}))
}

func TestRenderBadge_UnknownState(t *testing.T) {
	t.Parallel()
	out := renderBadge(integration.Field{Label: "State", Value: "something_else"})
	assert.Contains(t, xansi.Strip(out), "something_else")
}

func TestRenderHeaderRow_WithBadgesAndLinks(t *testing.T) {
	t.Parallel()
	badges := []integration.Field{
		{Label: "State", Value: "Triggered", Type: integration.FieldBadge},
	}
	links := []integration.Link{
		{Label: "Datadog", URL: "https://app.datadoghq.com/monitors/123"},
	}
	out := renderHeaderRow("Datadog", badges, links, nil)
	stripped := xansi.Strip(out)
	assert.Contains(t, stripped, "Datadog")
	assert.Contains(t, stripped, "Triggered")
}

func TestRenderHeaderRow_WithHyperlinks(t *testing.T) {
	t.Parallel()
	links := []integration.Link{
		{Label: "Console", URL: "https://example.com"},
	}
	a := ansi.Force()
	out := renderHeaderRow("GCP", nil, links, a)
	// OSC 8 sequence should be present.
	assert.Contains(t, out, "\x1b]8;;https://example.com")
}

func TestRenderHeaderRow_NoBadgesNoLinks(t *testing.T) {
	t.Parallel()
	out := renderHeaderRow("Unknown", nil, nil, nil)
	assert.Contains(t, xansi.Strip(out), "Unknown")
}

func TestRenderCodeBlock(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Query", Value: "avg(last_5m):avg:system.cpu > 90"}
	out := renderCodeBlock(f, 80)
	stripped := xansi.Strip(out)
	assert.Contains(t, stripped, "Query")
	assert.Contains(t, stripped, "avg(last_5m)")
}

func TestRenderCodeBlock_EmptyValue(t *testing.T) {
	t.Parallel()
	assert.Empty(t, renderCodeBlock(integration.Field{Label: "Query", Value: ""}, 80))
}

func TestRenderCodeBlock_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Q", Value: "x"}
	out := renderCodeBlock(f, 4)
	assert.NotEmpty(t, out)
}

func TestRenderMarkdownField(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Body", Value: "CPU is **high**"}
	out := renderMarkdownField(f, 80)
	assert.NotEmpty(t, out)
}

func TestRenderMarkdownField_EmptyValue(t *testing.T) {
	t.Parallel()
	assert.Empty(t, renderMarkdownField(integration.Field{Label: "Body", Value: ""}, 80))
}

func TestRenderTagPills(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Tags", Value: "service:api, env:prod, region:us-east-1"}
	out := renderTagPills(f, 200)
	stripped := xansi.Strip(out)
	assert.Contains(t, stripped, "service:api")
	assert.Contains(t, stripped, "env:prod")
	assert.Contains(t, stripped, "region:us-east-1")
}

func TestRenderTagPills_SingleTag(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Tags", Value: "env:prod"}
	out := renderTagPills(f, 200)
	assert.Contains(t, xansi.Strip(out), "env:prod")
}

func TestRenderTagPills_WrapsAtWidth(t *testing.T) {
	t.Parallel()
	f := integration.Field{Label: "Tags", Value: "service:payment-api, env:production, region:us-east-1, team:platform"}
	out := renderTagPills(f, 40)
	lines := len(splitLines(out))
	assert.Greater(t, lines, 1, "should wrap to multiple lines at narrow width")
}

func TestRenderTagPills_EmptyValue(t *testing.T) {
	t.Parallel()
	assert.Empty(t, renderTagPills(integration.Field{Label: "Tags", Value: ""}, 80))
}

func TestGroupFieldsByType(t *testing.T) {
	t.Parallel()
	fields := []integration.Field{
		{Label: "State", Value: "Triggered", Type: integration.FieldBadge},
		{Label: "Title", Value: "CPU High"},
		{Label: "Query", Value: "avg(...)", Type: integration.FieldCode},
		{Label: "Tags", Value: "env:prod", Type: integration.FieldTags},
		{Label: "Policy", Value: "Alert"},
	}
	groups := groupFieldsByType(fields)

	require.Len(t, groups[integration.FieldBadge], 1)
	assert.Equal(t, "State", groups[integration.FieldBadge][0].Label)

	require.Len(t, groups[integration.FieldText], 2)
	assert.Equal(t, "Title", groups[integration.FieldText][0].Label)
	assert.Equal(t, "Policy", groups[integration.FieldText][1].Label)

	require.Len(t, groups[integration.FieldCode], 1)
	require.Len(t, groups[integration.FieldTags], 1)
}

func TestGroupFieldsByType_EmptyInput(t *testing.T) {
	t.Parallel()
	groups := groupFieldsByType(nil)
	assert.Empty(t, groups)
}

// splitLines splits a string into lines, handling both \n and \r\n.
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
