package output

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// regexp2 (used by chroma for syntax highlighting) starts an internal
	// clock goroutine that outlives individual tests. Suppress the false positive.
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2.runClock"),
	)
}
