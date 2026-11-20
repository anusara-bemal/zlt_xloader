package api

import (
	"testing"
	"time"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

func TestIsFailure(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{`{"result":"failure"}`, true},
		{`{"result":"FAILURE"}`, true},
		{`{"result":"success"}`, false},
		{`<html>error</html>`, false},
		{`not json`, false},
	}
	for _, tt := range tests {
		if got := IsFailure([]byte(tt.body)); got != tt.want {
			t.Errorf("IsFailure(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

func TestMergePayloads(t *testing.T) {
	a := []profile.Payload{{ID: "A"}, {ID: "B"}}
	b := []profile.Payload{{ID: "B", Description: "overridden"}, {ID: "C"}}
	got := MergePayloads(a, b)
	if len(got) != 3 {
		t.Fatalf("got %d payloads: %+v", len(got), got)
	}
	if got[1].Description != "overridden" {
		t.Fatalf("later set did not win: %+v", got[1])
	}
}

func TestCSRFBypassDryRun(t *testing.T) {
	c := NewClient("192.0.2.1", time.Second, true, profile.Builtin().Endpoints)
	if err := c.CSRFBypass(); err != nil {
		t.Fatalf("CSRFBypass in dry-run: %v", err)
	}
	// md5 of the empty challenge (dry-run get is synthetic -> empty).
	if c.Token() == "" {
		t.Fatal("expected a derived token")
	}
}

func TestRunPayloadsDryRun(t *testing.T) {
	c := NewClient("192.0.2.1", time.Second, true, profile.Builtin().Endpoints)
	hits := c.RunPayloads(BuiltinPayloads(), 4, 0)
	if len(hits) != 0 {
		t.Fatalf("dry-run produced %d hits", len(hits))
	}
}
