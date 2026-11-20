//go:build integration

package outbound_test

import "fsos-server/_tests/testutil"

var (
	releaseBarrierOnCleanup = testutil.ReleaseBarrierOnCleanup
	awaitCacheSignal        = testutil.AwaitCacheSignal
)
