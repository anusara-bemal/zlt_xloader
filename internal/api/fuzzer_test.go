package api

import "testing"

func TestGenerateFuzzPayloads(t *testing.T) {
	payloads := GenerateFuzzPayloads(FuzzConfig{MaxRequests: 400})
	if len(payloads) == 0 {
		t.Fatal("no payloads generated")
	}
	if len(payloads) > 400 {
		t.Fatalf("generated %d payloads, exceeds cap", len(payloads))
	}

	seen := make(map[string]bool, len(payloads))
	hasUnlockID := false
	for _, p := range payloads {
		key := payloadKey(p.ID, p.Args)
		if seen[key] {
			t.Fatalf("duplicate payload: %s", key)
		}
		seen[key] = true
		if p.Args["isTest"] != "false" {
			t.Fatalf("%s: isTest = %q", p.ID, p.Args["isTest"])
		}
		if p.Method != "POST" {
			t.Fatalf("%s: method = %q", p.ID, p.Method)
		}
		if p.ID == "TZ_UNLOCK_PLMN" {
			hasUnlockID = true
		}
	}
	if !hasUnlockID {
		t.Fatal("expected verified unlock goformId in fuzz corpus")
	}
}

func TestGenerateFuzzPayloadsOverrides(t *testing.T) {
	payloads := GenerateFuzzPayloads(FuzzConfig{
		Keywords:    []string{"plmn"},
		GoformIDs:   []string{"CUSTOM_ID"},
		Values:      []string{"1"},
		MaxRequests: 100,
	})
	if len(payloads) == 0 {
		t.Fatal("no payloads generated")
	}
	found := false
	for _, p := range payloads {
		if p.ID == "CUSTOM_ID" {
			found = true
		}
	}
	if !found {
		t.Fatal("custom goformId not present")
	}
}
