package main

import (
	"strings"
	"testing"
)

// TestEmptyAPIListRefused checks that an explicitly empty --http.api or --ws.api stops geth. An
// empty module list registers every namespace, admin and debug included, so an empty value was
// once a one-character way to expose them.
func TestEmptyAPIListRefused(t *testing.T) {
	for _, flag := range []string{"--http.api", "--ws.api"} {
		geth := runGeth(t, flag, "", "dumpconfig")
		geth.WaitExit()
		if status, stderr := geth.ExitStatus(), geth.StderrText(); status == 0 || !strings.Contains(stderr, flag+" is empty") {
			t.Errorf("%s \"\": exit status %d, stderr %q; want it refused", flag, status, stderr)
		}
	}
}
