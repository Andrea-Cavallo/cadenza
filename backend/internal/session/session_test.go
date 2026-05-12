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
