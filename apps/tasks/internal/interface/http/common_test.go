package http_test

import "time"

// fixedNow はテスト用の固定時刻を返すヘルパー関数。
// すべてのテストで一貫した時刻を使用することで、テストの再現性を確保する。
func fixedNow() time.Time {
	return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
}
