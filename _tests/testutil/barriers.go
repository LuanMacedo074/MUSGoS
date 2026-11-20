package testutil

import (
	"testing"
	"time"
)

// ReleaseBarrierOnCleanup unblocks a paused operation even when an assertion ends
// the test before release. Tests close release only from the test goroutine.
func ReleaseBarrierOnCleanup(t testing.TB, release chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
}

// AwaitCacheSignal waits for an operation to reach its observed cache boundary.
func AwaitCacheSignal(t testing.TB, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not reach its cache boundary")
	}
}
