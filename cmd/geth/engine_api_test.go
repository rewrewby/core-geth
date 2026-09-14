// Copyright 2026 The core-geth Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mordor never merges, so its node starts no engine API. Starting one writes the JWT secret and logs
// "Engine API enabled", so either shows that it started. Engine API flags given to such a node are
// reported as ignored, and without them the node says nothing about the engine API at all.
func TestEngineAPINotStartedWithoutMerge(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		flags []string
	}{
		{name: "no engine API flags"},
		{name: "engine API flag given", flags: []string{"--authrpc.port", "0"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			datadir := t.TempDir()
			args := append([]string{"--mordor", "--datadir", datadir, "--syncmode=full", "--cache", "16",
				"--maxpeers", "0", "--port", "0", "--nodiscover", "--nat", "none", "--ipcdisable"}, tt.flags...)
			geth := runGeth(t, append(args, "--exec", "1", "console")...)
			geth.WaitExit()
			stderr := geth.StderrText()

			if _, err := os.Stat(filepath.Join(datadir, "geth", "jwtsecret")); err == nil {
				t.Error("JWT secret written: the engine API started")
			}
			if strings.Contains(stderr, "Engine API enabled") {
				t.Error("engine API logged as enabled")
			}
			ignored := strings.Contains(stderr, "Engine API not started") && strings.Contains(stderr, "ignored=--authrpc.port")
			if given := len(tt.flags) > 0; ignored != given {
				t.Errorf("--authrpc.port reported as ignored: %v, want %v\n%s", ignored, given, stderr)
			}
			if len(tt.flags) == 0 && strings.Contains(stderr, "Engine API") {
				t.Errorf("engine API mentioned without its flags\n%s", stderr)
			}
		})
	}
}
