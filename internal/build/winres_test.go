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
	"bytes"
	"debug/pe"
	"encoding/binary"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// The resource object must survive linking: this builds a Windows executable with it and reads the version
// information and manifest back out of the executable.
func TestWindowsResourceObject(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a Windows executable")
	}
	gotool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go tool on PATH")
	}
	dir := t.TempDir()
	want := [][2]string{{"CompanyName", "Example Company LLC"}, {"ProductVersion", "2.3.4-RC1"}, {"LegalCopyright", "Copyright © Example"}}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	res := WindowsResources{Name: "Example.Division.App", Major: 2, Minor: 3, Patch: 4, Prerelease: true, Strings: want}
	if err := WriteWindowsResourceObject(filepath.Join(dir, "zz_resources_windows_amd64.syso"), res); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(gotool, "build", "-o", "example.exe", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	f, err := pe.Open(filepath.Join(dir, "example.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	key, fixed, children := readVersionBlock(t, readResource(t, f, 16))
	if key != "VS_VERSION_INFO" || len(fixed) != 52 || binary.LittleEndian.Uint32(fixed) != 0xFEEF04BD {
		t.Fatalf("version information header: key %q, fixed length %d", key, len(fixed))
	}
	if ms, ls, flags := binary.LittleEndian.Uint32(fixed[8:]), binary.LittleEndian.Uint32(fixed[12:]), binary.LittleEndian.Uint32(fixed[28:]); ms != 2<<16|3 || ls != 4<<16 || flags != 0x2 {
		t.Errorf("fixed file information: version %#x %#x, flags %#x", ms, ls, flags)
	}
	var got [][2]string
	for _, child := range children {
		if key, _, tables := readVersionBlock(t, child); key == "StringFileInfo" {
			for _, table := range tables {
				_, _, entries := readVersionBlock(t, table)
				for _, entry := range entries {
					k, v, _ := readVersionBlock(t, entry)
					got = append(got, [2]string{k, decodeUTF16(v)})
				}
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("strings: got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("string %d: got %q, want %q", i, got[i], want[i])
		}
	}
	checkManifest(t, readResource(t, f, 24))
}

// The manifest must parse, open with the application's identity as the reference requires, and keep
// the executable at the caller's privilege level.
func checkManifest(t *testing.T, manifest []byte) {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(manifest))
	var elements []xml.StartElement
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("manifest does not parse: %v\n%s", err, manifest)
		}
		if se, ok := tok.(xml.StartElement); ok {
			elements = append(elements, se)
		}
	}
	if len(elements) < 2 || elements[0].Name.Local != "assembly" || elements[1].Name.Local != "assemblyIdentity" {
		t.Fatalf("manifest does not open with assembly and assemblyIdentity: %s", manifest)
	}
	identity := map[string]string{}
	for _, a := range elements[1].Attr {
		identity[a.Name.Local] = a.Value
	}
	if identity["type"] != "win32" || identity["name"] != "Example.Division.App" || identity["version"] != "2.3.4.0" || identity["processorArchitecture"] != "amd64" {
		t.Errorf("assembly identity: %v", identity)
	}
	var level string
	for _, se := range elements {
		if se.Name.Local == "requestedExecutionLevel" {
			for _, a := range se.Attr {
				if a.Name.Local == "level" {
					level = a.Value
				}
			}
		}
	}
	if level != "asInvoker" {
		t.Errorf("requested execution level %q, want asInvoker", level)
	}
}

func TestWindowsResourceObjectRejectsName(t *testing.T) {
	for _, name := range []string{"", "geth", "Example Company.App", `Example.App"/>`} {
		if err := WriteWindowsResourceObject(filepath.Join(t.TempDir(), "x.syso"), WindowsResources{Name: name}); err == nil {
			t.Errorf("name %q accepted", name)
		}
	}
}

// readResource returns the data of the first ID and language under a resource type.
func readResource(t *testing.T, f *pe.File, typ uint32) []byte {
	t.Helper()
	oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		t.Fatal("not a PE32+ executable")
	}
	rva := oh.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	for _, s := range f.Sections {
		if rva < s.VirtualAddress || rva >= s.VirtualAddress+s.VirtualSize {
			continue
		}
		data, err := s.Data()
		if err != nil {
			t.Fatal(err)
		}
		root := data[rva-s.VirtualAddress:]
		entry := func(dir []byte, i int) (uint32, uint32) {
			return binary.LittleEndian.Uint32(dir[16+8*i:]), binary.LittleEndian.Uint32(dir[20+8*i:])
		}
		count := int(binary.LittleEndian.Uint16(root[12:]) + binary.LittleEndian.Uint16(root[14:]))
		for i := 0; i < count; i++ {
			if id, off := entry(root, i); id == typ {
				_, off = entry(root[off&^0x80000000:], 0)
				_, off = entry(root[off&^0x80000000:], 0)
				dataRVA, size := binary.LittleEndian.Uint32(root[off:]), binary.LittleEndian.Uint32(root[off+4:])
				return data[dataRVA-s.VirtualAddress : dataRVA-s.VirtualAddress+size]
			}
		}
		t.Fatalf("no resource of type %d", typ)
	}
	t.Fatal("no resource section")
	return nil
}

// readVersionBlock parses one version information node, failing on a length that overruns its data.
func readVersionBlock(t *testing.T, b []byte) (key string, value []byte, children [][]byte) {
	t.Helper()
	length, valueLength, typ := int(binary.LittleEndian.Uint16(b)), int(binary.LittleEndian.Uint16(b[2:])), binary.LittleEndian.Uint16(b[4:])
	if length < 6 || length > len(b) {
		t.Fatalf("block length %d overruns %d bytes", length, len(b))
	}
	b = b[:length]
	i := 6
	var units []uint16
	for ; binary.LittleEndian.Uint16(b[i:]) != 0; i += 2 {
		units = append(units, binary.LittleEndian.Uint16(b[i:]))
	}
	i = (i + 2 + 3) &^ 3
	if typ == 1 {
		valueLength *= 2
	}
	if i+valueLength > len(b) {
		t.Fatalf("value of %q overruns its block", string(utf16.Decode(units)))
	}
	value = b[i : i+valueLength]
	for i = (i + valueLength + 3) &^ 3; i < len(b); {
		n := int(binary.LittleEndian.Uint16(b[i:]))
		if n < 6 || i+n > len(b) {
			t.Fatalf("child length %d overruns its parent", n)
		}
		children = append(children, b[i:i+n])
		i = (i + n + 3) &^ 3
	}
	return string(utf16.Decode(units)), value, children
}

func decodeUTF16(b []byte) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return strings.TrimRight(string(utf16.Decode(units)), "\x00")
}
