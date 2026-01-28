package main

import (
	"errors"
	"log"
)

const placeholderSecret = "default-secret-change-in-production"

// #nosec G101 -- これは本物のクレデンシャルではなく、開発環境のみで使用されるプレースホルダー
const devDefaultSecret = "dev-only-secret-change-me"

// resolveCursorSecret resolves CURSOR_SECRET based on environment.
// In production (APP_ENV=production), an empty or placeholder secret causes an error.
// In dev/test, it falls back to devDefaultSecret with a warning.
func resolveCursorSecret(appEnv string, raw string) ([]byte, error) {
	isProduction := appEnv == "production"

	if isProduction {
		if raw == "" {
			return nil, errors.New("CURSOR_SECRET must be set in production")
		}
		if raw == placeholderSecret {
			return nil, errors.New("CURSOR_SECRET must not be the placeholder value in production")
		}
		return []byte(raw), nil
	}

	// dev / test environment
	if raw == "" {
		log.Println("WARNING: CURSOR_SECRET is not set, using dev default secret (not for production)")
		return []byte(devDefaultSecret), nil
	}

	return []byte(raw), nil
}
