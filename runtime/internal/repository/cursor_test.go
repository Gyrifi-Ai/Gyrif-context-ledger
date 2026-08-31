package repository

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, time.August, 31, 12, 34, 56, 123456789, time.UTC)
	encoded := EncodeCursor(createdAt, "chg_123")
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.CreatedAt.Equal(createdAt) || decoded.ID != "chg_123" {
		t.Fatalf("decoded cursor = %#v", decoded)
	}
	if encoded[len(encoded)-1] == '=' {
		t.Fatalf("cursor has padding: %q", encoded)
	}
}

func TestDecodeCursorRejectsMalformedValues(t *testing.T) {
	values := []string{
		"",
		"not-base64!",
		base64.RawURLEncoding.EncodeToString([]byte(`not json`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-08-31T00:00:00Z"}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"not-a-time","i":"chg_1"}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-08-31T01:00:00+01:00","i":"chg_1"}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-08-31T00:00:00Z","i":" "}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-08-31T00:00:00Z","i":"chg_1","extra":true}`)),
		base64.RawURLEncoding.EncodeToString([]byte(`{"t":"2026-08-31T00:00:00Z","i":"chg_1"}{}`)),
	}
	for _, value := range values {
		if _, err := DecodeCursor(value); err == nil {
			t.Errorf("DecodeCursor(%q) succeeded", value)
		}
	}
}
