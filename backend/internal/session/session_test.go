package session_test

import (
	"testing"

	"github.com/Andrea-Cavallo/cadenza/internal/session"
)

func TestSaveReasonString(t *testing.T) {
	cases := []struct {
		r    session.SaveReason
		want string
	}{
		{session.SaveReasonManual, "manual"},
		{session.SaveReasonAuto, "auto"},
		{session.SaveReasonEvict, "evict"},
		{session.SaveReasonShutdown, "shutdown"},
		{session.SaveReason(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("SaveReason(%d).String() = %q, want %q", tc.r, got, tc.want)
		}
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	original := session.ExportedHeader{
		SaveReason:   uint8(session.SaveReasonAuto),
		MessageCount: 42,
		PatternCount: 7,
		CreatedAt:    1715000000,
		UpdatedAt:    1715001000,
	}
	encoded := session.EncodeHeader(original)
	got, err := session.DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if got != original {
		t.Errorf("round-trip failed: got %+v, want %+v", got, original)
	}
}

func TestDecodeHeader_InvalidMagic(t *testing.T) {
	var buf [32]byte
	buf[0], buf[1], buf[2] = 'X', 'X', 'X'
	if _, err := session.DecodeHeader(buf); err == nil {
		t.Error("expected error for invalid magic, got nil")
	}
}

func TestDecodeHeader_InvalidVersion(t *testing.T) {
	var buf [32]byte
	buf[0], buf[1], buf[2] = 'C', 'Z', 'S'
	buf[3] = 99
	if _, err := session.DecodeHeader(buf); err == nil {
		t.Error("expected error for invalid version, got nil")
	}
}
