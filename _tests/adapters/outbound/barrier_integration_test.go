//go:build integration

package outbound_test

import "musgos/_tests/testutil"

var (
	releaseBarrierOnCleanup = testutil.ReleaseBarrierOnCleanup
	awaitCacheSignal        = testutil.AwaitCacheSignal
)
