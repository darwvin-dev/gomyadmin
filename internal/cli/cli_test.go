package cli

import "testing"

func TestUnknownCommandReturnsError(t *testing.T) {
	if code := Run([]string{"missing"}); code == 0 {
		t.Fatal("expected non-zero exit code")
	}
}
