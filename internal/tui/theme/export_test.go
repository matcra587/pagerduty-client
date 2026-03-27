package theme

import "sync"

// ResetForTest resets the sync.Once guard so Apply can be called
// again in tests. Theme tests must not use t.Parallel because
// Apply mutates package-level variables.
func ResetForTest() {
	applyOnce = sync.Once{}
}
