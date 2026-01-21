package task

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CursorPayload は cursor の payload を表す。
type CursorPayload struct {
	V         int    `json:"v"`
	CreatedAt string `json:"createdAt"` // RFC3339Nanoだが **micro秒精度**
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	QHash     string `json:"qhash"`
	IssuedAt  int64  `json:"iat"`
}

// EncodeCursor は cursor をエンコードする。
// payload(JSON) → base64.RawURLEncoding（paddingなし） = encodedPayload
// sig = HMAC-SHA256(secret, encodedPayload) → base64.RawURLEncoding
// cursor = encodedPayload + "." + sig
func EncodeCursor(payload CursorPayload, secret []byte) (string, error) {
	// payload を JSON に変換
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// base64.RawURLEncoding でエンコード（paddingなし）
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// HMAC-SHA256 で署名
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(encodedPayload))
	sig := mac.Sum(nil)

	// 署名を base64.RawURLEncoding でエンコード
	encodedSig := base64.RawURLEncoding.EncodeToString(sig)

	// cursor = encodedPayload + "." + sig
	return encodedPayload + "." + encodedSig, nil
}

// DecodeCursor は cursor をデコードし、署名を検証する。
// エラーは validation error として返す（500にしない）。
func DecodeCursor(cursorStr string, secret []byte) (*CursorPayload, error) {
	// フォーマットチェック: "payload.sig" の形式
	parts := strings.Split(cursorStr, ".")
	if len(parts) != 2 {
		return nil, ErrCursorInvalidFormat
	}

	encodedPayload := parts[0]
	encodedSig := parts[1]

	// payload をデコード
	payloadJSON, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, ErrCursorInvalidFormat
	}

	// JSON をパース
	var payload CursorPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, ErrCursorInvalidFormat
	}

	// 署名を検証
	expectedSig, err := base64.RawURLEncoding.DecodeString(encodedSig)
	if err != nil {
		return nil, ErrCursorInvalidFormat
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(encodedPayload))
	computedSig := mac.Sum(nil)

	if !hmac.Equal(expectedSig, computedSig) {
		return nil, ErrCursorInvalidSignature
	}

	return &payload, nil
}

// ParseCursorCreatedAt は cursor の createdAt 文字列を time.Time に変換し、micro秒に丸める。
func ParseCursorCreatedAt(createdAtStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, createdAtStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cursor format: %w", err)
	}
	// micro秒に丸める
	return t.Truncate(time.Microsecond), nil
}

// FormatCursorCreatedAt は time.Time を RFC3339Nano 形式の文字列に変換する（micro秒精度）。
func FormatCursorCreatedAt(t time.Time) string {
	// micro秒に丸めてからフォーマット
	return t.Truncate(time.Microsecond).Format(time.RFC3339Nano)
}

// ValidateCursorExpiry は cursor の有効期限をチェックする（24時間）。
// 期限切れの場合はエラーを返す。
func ValidateCursorExpiry(payload *CursorPayload, now time.Time) error {
	nowUnix := now.Unix()
	if nowUnix-payload.IssuedAt > 86400 {
		return ErrCursorExpired
	}
	return nil
}
