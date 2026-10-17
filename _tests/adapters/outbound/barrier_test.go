package outbound_test

import "testing"

// releaseBarrierOnCleanup unblocks a paused cache barrier when a test fails
// before it got there, so the execution under test can never outlive the test.
func releaseBarrierOnCleanup(t *testing.T, release chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
}
