package safety

import "testing"

func TestConfirmYesFlag(t *testing.T) {
	if !Confirm("proceed?", true) {
		t.Fatal("--yes should skip the prompt")
	}
}
