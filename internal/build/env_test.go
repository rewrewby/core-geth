// Copyright 2026 The core-geth Authors
// This file is part of the core-geth library.
//
// The core-geth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The core-geth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the core-geth library. If not, see <http://www.gnu.org/licenses/>.

package build

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// A release tag is built from a detached checkout, and its binaries must still carry the commit date.
func TestLocalEnvDetachedHeadDate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on PATH")
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE=2026-09-13T12:00:00Z", "GIT_AUTHOR_DATE=2026-09-13T12:00:00Z")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("commit", "-q", "--allow-empty", "-m", "release")
	commit := git("rev-parse", "HEAD")
	git("checkout", "-q", "--detach", commit)
	t.Chdir(dir)

	env := LocalEnv()
	if env.Commit != commit {
		t.Fatalf("commit %q, want %q", env.Commit, commit)
	}
	if env.Date != "20260913" {
		t.Errorf("date %q, want 20260913", env.Date)
	}
}
