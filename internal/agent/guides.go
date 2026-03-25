package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed guides
var guidesFS embed.FS

// GuideNames lists available guide names, derived from embedded filenames.
var GuideNames []string

func init() {
	entries, err := fs.ReadDir(guidesFS, "guides")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if base, ok := strings.CutSuffix(name, ".md"); ok {
			GuideNames = append(GuideNames, base)
		}
	}
}

// Guide returns the content of the named agent guide.
func Guide(name string) (string, error) {
	path := "guides/" + name + ".md"

	b, err := guidesFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("guide %q not found (available: %s)", name, strings.Join(GuideNames, ", "))
	}

	return string(b), nil
}
