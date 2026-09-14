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
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"unicode/utf16"
)

// WindowsResources is the version information and application manifest of a Windows executable. Windows
// shows the version information in the file's properties and in Task Manager.
type WindowsResources struct {
	Name                string // the manifest's application identity, as Organization.Division.Name
	Major, Minor, Patch uint16
	Prerelease          bool
	Strings             [][2]string // name and value pairs, in the order written
}

var manifestName = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)+$`)

// manifest identifies the application and its version, runs it at the caller's privilege level, and
// declares support for Windows 10, whose ID also covers Windows 11 and Windows Server 2016 and later. Go
// supports no earlier Windows. Long paths need no entry, because the Go runtime enables them itself.
func (res WindowsResources) manifest() []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="%s" version="%d.%d.%d.0" processorArchitecture="amd64"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
    </application>
  </compatibility>
</assembly>
`, res.Name, res.Major, res.Minor, res.Patch))
}

// WriteWindowsResourceObject writes res as a COFF object for windows/amd64. Placed in a main package's
// directory under a name ending in _windows_amd64.syso, it is linked into that package's executable, by
// Go's linker or by an external one.
func WriteWindowsResourceObject(path string, res WindowsResources) error {
	// Windows refuses to start an executable whose manifest it cannot parse, so reject a name that is
	// not a plain dotted identifier rather than escape it.
	if !manifestName.MatchString(res.Name) {
		return fmt.Errorf("manifest name %q is not of the form Organization.Division.Name", res.Name)
	}
	// Resource types are listed in ascending order, as the format requires.
	section, relocs := resourceSection([]resource{
		{typ: 16, id: 1, lang: 0x0409, data: res.versionInfo()}, // RT_VERSION
		{typ: 24, id: 1, lang: 0x0409, data: res.manifest()},    // RT_MANIFEST
	})
	const fileHeaderSize, sectionHeaderSize, relocSize = 20, 40, 10
	rawStart := uint32(fileHeaderSize + sectionHeaderSize)
	relocStart := rawStart + uint32(len(section))
	symbolStart := relocStart + uint32(len(relocs))*relocSize

	var obj bytes.Buffer
	w := func(v any) { binary.Write(&obj, binary.LittleEndian, v) }
	w(uint16(0x8664)) // IMAGE_FILE_MACHINE_AMD64
	w(uint16(1))      // sections
	w(uint32(0))      // timestamp: none, so the build stays reproducible
	w(symbolStart)
	w(uint32(1)) // symbols
	w(uint16(0)) // optional header size
	w(uint16(0)) // characteristics

	obj.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0)) // virtual size
	w(uint32(0)) // virtual address
	w(uint32(len(section)))
	w(rawStart)
	w(relocStart)
	w(uint32(0)) // line numbers
	w(uint16(len(relocs)))
	w(uint16(0))          // line numbers
	w(uint32(0x40000040)) // IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
	obj.Write(section)

	// Each data entry holds its data's offset in the section. Relocating it against the section symbol
	// turns that into the address the linker places it at.
	for _, off := range relocs {
		w(off)
		w(uint32(0)) // symbol 0, the section
		w(uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
	}
	obj.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0)) // value
	w(int16(1))  // section number
	w(uint16(0)) // type
	w(uint8(3))  // IMAGE_SYM_CLASS_STATIC
	w(uint8(0))  // auxiliary symbols
	w(uint32(4)) // string table: its own size field only
	return os.WriteFile(path, obj.Bytes(), 0644)
}

type resource struct {
	typ, id, lang uint32
	data          []byte
}

// resourceSection lays out a resource section in which every type holds one ID in one language, and returns
// the section with the offsets of the fields that need relocating.
func resourceSection(resources []resource) ([]byte, []uint32) {
	const dirSize, entrySize, dataEntrySize = 16, 8, 16
	n := uint32(len(resources))
	leafDirSize := uint32(dirSize + entrySize)
	idDirs := dirSize + n*entrySize
	langDirs := idDirs + n*leafDirSize
	dataEntries := langDirs + n*leafDirSize

	var buf bytes.Buffer
	w := func(v any) { binary.Write(&buf, binary.LittleEndian, v) }
	dir := func(entries uint16) {
		w(uint32(0))             // characteristics
		w(uint32(0))             // timestamp
		w([2]uint16{})           // version
		w([2]uint16{0, entries}) // named entries, ID entries
	}
	const subdirectory = 0x80000000
	dir(uint16(n))
	for i, r := range resources {
		w([2]uint32{r.typ, subdirectory | (idDirs + uint32(i)*leafDirSize)})
	}
	for i, r := range resources {
		dir(1)
		w([2]uint32{r.id, subdirectory | (langDirs + uint32(i)*leafDirSize)})
	}
	for i, r := range resources {
		dir(1)
		w([2]uint32{r.lang, dataEntries + uint32(i)*dataEntrySize})
	}
	offsets := make([]uint32, n)
	next := dataEntries + n*dataEntrySize
	for i, r := range resources {
		next = (next + 7) &^ 7
		offsets[i] = next
		next += uint32(len(r.data))
	}
	relocs := make([]uint32, n)
	for i, r := range resources {
		relocs[i] = uint32(buf.Len())
		w([4]uint32{offsets[i], uint32(len(r.data)), 0, 0}) // data, size, code page, reserved
	}
	for i, r := range resources {
		for uint32(buf.Len()) < offsets[i] {
			buf.WriteByte(0)
		}
		buf.Write(r.data)
	}
	return buf.Bytes(), relocs
}

// versionInfo encodes res as a VS_VERSIONINFO tree, in U.S. English with Unicode strings.
func (res WindowsResources) versionInfo() []byte {
	ms := uint32(res.Major)<<16 | uint32(res.Minor)
	ls := uint32(res.Patch) << 16
	var flags uint32
	if res.Prerelease {
		flags = 0x2 // VS_FF_PRERELEASE
	}
	var fixed bytes.Buffer
	binary.Write(&fixed, binary.LittleEndian, [13]uint32{
		0xFEEF04BD, 0x00010000, // signature, structure version
		ms, ls, ms, ls, // file version, product version
		0x3F, flags,
		0x00040004, // VOS_NT_WINDOWS32
		1, 0,       // VFT_APP, no subtype
		0, 0, // date
	})
	var strs [][]byte
	for _, kv := range res.Strings {
		value := utf16z(kv[1])
		strs = append(strs, versionBlock(kv[0], 1, value, uint16(len(value)/2)))
	}
	translation := []byte{0x09, 0x04, 0xB0, 0x04} // U.S. English, Unicode
	return versionBlock("VS_VERSION_INFO", 0, fixed.Bytes(), uint16(fixed.Len()),
		versionBlock("StringFileInfo", 1, nil, 0, versionBlock("040904B0", 1, nil, 0, strs...)),
		versionBlock("VarFileInfo", 1, nil, 0, versionBlock("Translation", 0, translation, uint16(len(translation)))),
	)
}

// versionBlock encodes one node of a version information tree. Its value and each child start on a
// four-byte boundary, where Windows looks for them.
func versionBlock(key string, typ uint16, value []byte, valueLength uint16, children ...[]byte) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [3]uint16{0, valueLength, typ})
	b.Write(utf16z(key))
	pad4(&b)
	b.Write(value)
	for _, child := range children {
		pad4(&b)
		b.Write(child)
	}
	out := b.Bytes()
	binary.LittleEndian.PutUint16(out, uint16(len(out)))
	return out
}

func utf16z(s string) []byte {
	units := utf16.Encode([]rune(s + "\x00"))
	b := make([]byte, 2*len(units))
	for i, u := range units {
		binary.LittleEndian.PutUint16(b[2*i:], u)
	}
	return b
}

func pad4(b *bytes.Buffer) {
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
}
