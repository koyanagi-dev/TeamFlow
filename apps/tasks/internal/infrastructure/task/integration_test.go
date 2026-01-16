//go:build integration

package taskinfra

import (
	"os"
	"testing"

	"teamflow-tasks/internal/testutil"
)

func TestMain(m *testing.M) {
	code := testutil.InitTestDB(m)
	if code != 0 {
		// InitTestDB already printed error messages
		os.Exit(code)
	}
	// Set testPool to the initialized pool
	testPool = testutil.TestPool
	os.Exit(code)
}
