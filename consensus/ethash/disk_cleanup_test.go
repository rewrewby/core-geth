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

package ethash

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The cleanup after writing a cache or DAG to disk must keep the file it just wrote. Its lower bound,
// the epoch minus the on-disk limit, is unsigned: below the limit it wrapped around, every file matched,
// and the file still mapped in memory was deleted (Linux, macOS) or refused with an error (Windows).
func TestDiskCleanupKeepsCurrentEpoch(t *testing.T) {
	const limit, epochLength = 3, 30000
	kinds := []struct {
		prefix   string
		generate func(dir string, epoch uint64)
	}{
		{"cache", func(dir string, epoch uint64) { newCache(epoch, epochLength).generate(dir, limit, false, true) }},
		{"full", func(dir string, epoch uint64) { newDataset(epoch, epochLength).generate(dir, limit, false, true) }},
	}
	for _, kind := range kinds {
		for _, epoch := range []uint64{0, 1, limit - 1, limit, 10} {
			dir := t.TempDir()
			file := func(e uint64) string {
				return filepath.Join(dir, fmt.Sprintf("%s-R%d-%d-%016x", kind.prefix, algorithmRevision, e, 0))
			}
			exists := func(path string) bool { _, err := os.Stat(path); return err == nil }

			// Files outside the window, which the cleanup must still remove.
			ahead := file(epoch + 5)
			os.WriteFile(ahead, nil, 0600)
			var behind string
			if epoch >= limit {
				behind = file(epoch - limit)
				os.WriteFile(behind, nil, 0600)
			}
			kind.generate(dir, epoch)

			current, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("%s-R%d-%d-*", kind.prefix, algorithmRevision, epoch)))
			if len(current) != 1 {
				t.Errorf("%s epoch %d: the file just written is gone from disk", kind.prefix, epoch)
			}
			if exists(ahead) {
				t.Errorf("%s epoch %d: file for epoch %d not removed", kind.prefix, epoch, epoch+5)
			}
			if behind != "" && exists(behind) {
				t.Errorf("%s epoch %d: file for epoch %d not removed", kind.prefix, epoch, epoch-limit)
			}
		}
	}
}
