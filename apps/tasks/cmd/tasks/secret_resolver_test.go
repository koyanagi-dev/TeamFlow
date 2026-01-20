package main

import (
	"bytes"
	"testing"
)

func TestResolveCursorSecret(t *testing.T) {
	tests := []struct {
		name        string
		appEnv      string
		rawSecret   string
		wantSecret  []byte
		wantErr     bool
	}{
		// production cases
		{
			name:       "production with empty secret should fail",
			appEnv:     "production",
			rawSecret:  "",
			wantSecret: nil,
			wantErr:    true,
		},
		{
			name:       "production with placeholder secret should fail",
			appEnv:     "production",
			rawSecret:  "default-secret-change-in-production",
			wantSecret: nil,
			wantErr:    true,
		},
		{
			name:       "production with valid secret should succeed",
			appEnv:     "production",
			rawSecret:  "valid-secret",
			wantSecret: []byte("valid-secret"),
			wantErr:    false,
		},

		// dev / test cases
		{
			name:       "empty APP_ENV with empty secret should use dev default",
			appEnv:     "",
			rawSecret:  "",
			wantSecret: []byte(devDefaultSecret),
			wantErr:    false,
		},
		{
			name:       "development with empty secret should use dev default",
			appEnv:     "development",
			rawSecret:  "",
			wantSecret: []byte(devDefaultSecret),
			wantErr:    false,
		},
		{
			name:       "test with empty secret should use dev default",
			appEnv:     "test",
			rawSecret:  "",
			wantSecret: []byte(devDefaultSecret),
			wantErr:    false,
		},
		{
			name:       "development with custom secret should use custom secret",
			appEnv:     "development",
			rawSecret:  "custom-secret",
			wantSecret: []byte("custom-secret"),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSecret, err := resolveCursorSecret(tt.appEnv, tt.rawSecret)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !bytes.Equal(gotSecret, tt.wantSecret) {
				t.Errorf("got secret %q, want %q", gotSecret, tt.wantSecret)
			}
		})
	}
}
