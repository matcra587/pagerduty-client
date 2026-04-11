package integration

import (
	"fmt"
	"strings"

	"github.com/matcra587/pagerduty-client/internal/config"
)

// ResolveExternalLink resolves the best external link from an alert
// body, checking user-configured custom fields first, then falling
// back to integration-detected links. Returns "" when no link is
// found.
func ResolveExternalLink(cfg *config.Config, body map[string]any) string {
	if body == nil {
		return ""
	}

	if cfg != nil {
		for _, cf := range cfg.CustomFields {
			if cf.Display != "link" {
				continue
			}
			val, ok := ResolveFieldPath(body, cf.Path)
			if !ok {
				continue
			}
			s := fmt.Sprintf("%v", val)
			if s != "" && s != "<nil>" && IsHTTP(s) {
				return s
			}
		}
	}

	summary := Detect(body)
	if len(summary.Links) > 0 && IsHTTP(summary.Links[0].URL) {
		return summary.Links[0].URL
	}

	return ""
}

// IsHTTP reports whether url starts with http:// or https://.
func IsHTTP(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// ResolveFieldPath walks the body map at the given dot-separated path.
// If the path is not found at the top level, it falls back to looking
// inside cef_details.
func ResolveFieldPath(body map[string]any, path string) (any, bool) {
	if val, ok := walkPath(body, path); ok {
		return val, true
	}
	if cef, ok := body["cef_details"].(map[string]any); ok {
		return walkPath(cef, path)
	}
	return nil, false
}

func walkPath(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = root

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}
