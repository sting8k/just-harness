package cli

import (
	"bytes"
	"testing"
)

func TestVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"version"}, &out, &errb); code != 0 || out.String() != "dev\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errb.String())
	}
}
