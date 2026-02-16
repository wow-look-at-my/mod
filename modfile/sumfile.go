// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package modfile

import (
	"fmt"
	"strings"

	"github.com/wow-look-at-my/mod/module"
)

// A SumFile is the parsed, interpreted form of a go.sum file.
type SumFile struct {
	Hash []*Hash
}

// A Hash is a single hash entry in a go.sum file.
// Each entry gives the hash for a specific module version,
// either the hash of the module's zip file or the hash of the
// module's go.mod file (indicated by a "/go.mod" suffix on the version).
type Hash struct {
	Mod    module.Version
	Hash   string
	GoMod  bool // whether this is a go.mod hash (version has /go.mod suffix)
	Syntax SumLine
}

// A SumLine represents a single line in a go.sum file,
// recording its original position for formatting purposes.
type SumLine struct {
	Path    string
	Version string
	Hash    string
	offset  int // byte offset of line start in original file (-1 if added)
}

// ParseSum parses and returns a go.sum file.
//
// file is the name of the file, used in error messages.
//
// data is the content of the file.
func ParseSum(file string, data []byte) (*SumFile, error) {
	f := &SumFile{}
	var errs ErrorList

	text := string(data)
	lineno := 0
	offset := 0
	for len(text) > 0 {
		lineno++
		var line string
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			line = text[:i]
			text = text[i+1:]
		} else {
			line = text
			text = ""
		}
		lineOffset := offset
		offset += len(line) + 1 // +1 for \n

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		f0, f1, f2, ok := parseSumLine(line)
		if !ok {
			errs = append(errs, Error{
				Filename: file,
				Pos:      Position{Line: lineno, LineRune: 1, Byte: lineOffset},
				Err:      fmt.Errorf("malformed go.sum line: %s", line),
			})
			continue
		}

		// Determine if this is a /go.mod hash line.
		gomod := false
		version := f1
		if trimmed, ok := strings.CutSuffix(f1, "/go.mod"); ok {
			gomod = true
			version = trimmed
		}

		f.Hash = append(f.Hash, &Hash{
			Mod:   module.Version{Path: f0, Version: version},
			Hash:  f2,
			GoMod: gomod,
			Syntax: SumLine{
				Path:    f0,
				Version: f1,
				Hash:    f2,
				offset:  lineOffset,
			},
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return f, nil
}

// parseSumLine splits a go.sum line into its three fields:
// module path, version, and hash.
func parseSumLine(line string) (path, version, hash string, ok bool) {
	f := strings.Fields(line)
	if len(f) != 3 {
		return "", "", "", false
	}
	return f[0], f[1], f[2], true
}

// FormatSum returns the contents of the go.sum file as a byte slice,
// formatted in standard style. The output always ends with a newline
// if there are any entries.
func FormatSum(f *SumFile) []byte {
	var buf strings.Builder
	for _, h := range f.Hash {
		if h.Mod.Path == "" {
			continue // cleared entry
		}
		version := h.Mod.Version
		if h.GoMod {
			version += "/go.mod"
		}
		fmt.Fprintf(&buf, "%s %s %s\n", h.Mod.Path, version, h.Hash)
	}
	return []byte(buf.String())
}

// AddHash adds a new hash entry to the file.
// If an identical entry (same path, version, gomod flag, and hash) already exists,
// AddHash is a no-op.
func (f *SumFile) AddHash(mod module.Version, gomod bool, hash string) {
	for _, h := range f.Hash {
		if h.Mod == mod && h.GoMod == gomod && h.Hash == hash {
			return // already present
		}
	}
	version := mod.Version
	if gomod {
		version += "/go.mod"
	}
	f.Hash = append(f.Hash, &Hash{
		Mod:   mod,
		Hash:  hash,
		GoMod: gomod,
		Syntax: SumLine{
			Path:    mod.Path,
			Version: version,
			Hash:    hash,
			offset:  -1,
		},
	})
}

// DropHash removes all hash entries for the given module version.
// If gomod is true, only the /go.mod hash is removed.
// If gomod is false, only the non-/go.mod hash is removed.
func (f *SumFile) DropHash(mod module.Version, gomod bool) {
	for i := range f.Hash {
		if f.Hash[i].Mod == mod && f.Hash[i].GoMod == gomod {
			f.Hash[i].Mod.Path = "" // mark for cleanup
		}
	}
}

// DropAll removes all hash entries for the given module path and version,
// including both zip and go.mod hashes.
func (f *SumFile) DropAll(mod module.Version) {
	for i := range f.Hash {
		if f.Hash[i].Mod == mod {
			f.Hash[i].Mod.Path = "" // mark for cleanup
		}
	}
}

// Cleanup cleans up the file after edit operations.
// Modifications like DropHash clear the entry but do not remove it
// from the slice. Cleanup removes all cleared entries.
func (f *SumFile) Cleanup() {
	w := 0
	for _, h := range f.Hash {
		if h.Mod.Path != "" {
			f.Hash[w] = h
			w++
		}
	}
	f.Hash = f.Hash[:w]
}
