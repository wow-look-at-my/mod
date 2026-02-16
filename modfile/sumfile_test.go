// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package modfile

import (
	"testing"

	"github.com/wow-look-at-my/mod/module"
)

var parseSumTests = []struct {
	name    string
	in      string
	want    []*Hash
	wantErr bool
}{
	{
		name: "empty",
		in:   "",
		want: nil,
	},
	{
		name: "single_zip_hash",
		in:   "golang.org/x/text v0.3.0 h1:abc123=\n",
		want: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: false},
		},
	},
	{
		name: "single_gomod_hash",
		in:   "golang.org/x/text v0.3.0/go.mod h1:abc123=\n",
		want: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: true},
		},
	},
	{
		name: "multiple_entries",
		in: "golang.org/x/text v0.3.0 h1:abc123=\n" +
			"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
			"rsc.io/quote v1.5.2 h1:ghi789=\n" +
			"rsc.io/quote v1.5.2/go.mod h1:jkl012=\n",
		want: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: false},
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:def456=", GoMod: true},
			{Mod: module.Version{Path: "rsc.io/quote", Version: "v1.5.2"}, Hash: "h1:ghi789=", GoMod: false},
			{Mod: module.Version{Path: "rsc.io/quote", Version: "v1.5.2"}, Hash: "h1:jkl012=", GoMod: true},
		},
	},
	{
		name: "blank_lines_ignored",
		in: "\n" +
			"golang.org/x/text v0.3.0 h1:abc123=\n" +
			"\n" +
			"rsc.io/quote v1.5.2 h1:ghi789=\n" +
			"\n",
		want: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: false},
			{Mod: module.Version{Path: "rsc.io/quote", Version: "v1.5.2"}, Hash: "h1:ghi789=", GoMod: false},
		},
	},
	{
		name: "no_trailing_newline",
		in:   "golang.org/x/text v0.3.0 h1:abc123=",
		want: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: false},
		},
	},
	{
		name:    "malformed_line_too_few_fields",
		in:      "golang.org/x/text v0.3.0\n",
		wantErr: true,
	},
	{
		name:    "malformed_line_too_many_fields",
		in:      "golang.org/x/text v0.3.0 h1:abc123= extra\n",
		wantErr: true,
	},
}

func TestParseSum(t *testing.T) {
	for _, tt := range parseSumTests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseSum("go.sum", []byte(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Hash) != len(tt.want) {
				t.Fatalf("got %d entries, want %d", len(f.Hash), len(tt.want))
			}
			for i, got := range f.Hash {
				want := tt.want[i]
				if got.Mod != want.Mod {
					t.Errorf("entry %d: Mod = %v, want %v", i, got.Mod, want.Mod)
				}
				if got.Hash != want.Hash {
					t.Errorf("entry %d: Hash = %q, want %q", i, got.Hash, want.Hash)
				}
				if got.GoMod != want.GoMod {
					t.Errorf("entry %d: GoMod = %v, want %v", i, got.GoMod, want.GoMod)
				}
			}
		})
	}
}

func TestFormatSum(t *testing.T) {
	f := &SumFile{
		Hash: []*Hash{
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:abc123=", GoMod: false},
			{Mod: module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}, Hash: "h1:def456=", GoMod: true},
			{Mod: module.Version{Path: "rsc.io/quote", Version: "v1.5.2"}, Hash: "h1:ghi789=", GoMod: false},
		},
	}
	got := string(FormatSum(f))
	want := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n"
	if got != want {
		t.Errorf("FormatSum:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSumEmpty(t *testing.T) {
	f := &SumFile{}
	got := string(FormatSum(f))
	if got != "" {
		t.Errorf("FormatSum(empty) = %q, want %q", got, "")
	}
}

func TestParseSumRoundTrip(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n" +
		"rsc.io/quote v1.5.2/go.mod h1:jkl012=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	got := string(FormatSum(f))
	if got != in {
		t.Errorf("round trip:\ngot:\n%s\nwant:\n%s", got, in)
	}
}

func TestSumAddHash(t *testing.T) {
	f := &SumFile{}
	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}

	f.AddHash(mod, false, "h1:abc123=")
	if len(f.Hash) != 1 {
		t.Fatalf("after first AddHash: got %d entries, want 1", len(f.Hash))
	}

	// Adding the same entry again should be a no-op.
	f.AddHash(mod, false, "h1:abc123=")
	if len(f.Hash) != 1 {
		t.Fatalf("after duplicate AddHash: got %d entries, want 1", len(f.Hash))
	}

	// Adding a go.mod hash for the same module should add a new entry.
	f.AddHash(mod, true, "h1:def456=")
	if len(f.Hash) != 2 {
		t.Fatalf("after gomod AddHash: got %d entries, want 2", len(f.Hash))
	}

	// Adding a different hash for the same module+gomod should add a new entry.
	f.AddHash(mod, false, "h1:different=")
	if len(f.Hash) != 3 {
		t.Fatalf("after different hash AddHash: got %d entries, want 3", len(f.Hash))
	}

	got := string(FormatSum(f))
	want := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"golang.org/x/text v0.3.0 h1:different=\n"
	if got != want {
		t.Errorf("AddHash result:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSumDropHash(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	// Drop only the zip hash for golang.org/x/text.
	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}
	f.DropHash(mod, false)
	f.Cleanup()

	got := string(FormatSum(f))
	want := "golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n"
	if got != want {
		t.Errorf("DropHash(zip):\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSumDropHashGoMod(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	// Drop only the go.mod hash.
	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}
	f.DropHash(mod, true)
	f.Cleanup()

	got := string(FormatSum(f))
	want := "golang.org/x/text v0.3.0 h1:abc123=\n"
	if got != want {
		t.Errorf("DropHash(gomod):\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSumDropAll(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n" +
		"rsc.io/quote v1.5.2/go.mod h1:jkl012=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}
	f.DropAll(mod)
	f.Cleanup()

	got := string(FormatSum(f))
	want := "rsc.io/quote v1.5.2 h1:ghi789=\n" +
		"rsc.io/quote v1.5.2/go.mod h1:jkl012=\n"
	if got != want {
		t.Errorf("DropAll:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSumDropNonexistent(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	// Dropping a nonexistent entry should be a no-op.
	mod := module.Version{Path: "rsc.io/quote", Version: "v1.0.0"}
	f.DropAll(mod)
	f.Cleanup()

	got := string(FormatSum(f))
	if got != in {
		t.Errorf("DropAll(nonexistent):\ngot:\n%s\nwant:\n%s", got, in)
	}
}

func TestSumCleanup(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"golang.org/x/text v0.3.0/go.mod h1:def456=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	// Drop an entry, then check that Cleanup removes it.
	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}
	f.DropAll(mod)

	// Before cleanup, cleared entries still in slice.
	if len(f.Hash) != 3 {
		t.Fatalf("before Cleanup: got %d entries, want 3", len(f.Hash))
	}

	f.Cleanup()

	if len(f.Hash) != 1 {
		t.Fatalf("after Cleanup: got %d entries, want 1", len(f.Hash))
	}
	if f.Hash[0].Mod.Path != "rsc.io/quote" {
		t.Errorf("remaining entry path = %q, want %q", f.Hash[0].Mod.Path, "rsc.io/quote")
	}
}

func TestSumFormatSkipsClearedEntries(t *testing.T) {
	in := "golang.org/x/text v0.3.0 h1:abc123=\n" +
		"rsc.io/quote v1.5.2 h1:ghi789=\n"

	f, err := ParseSum("go.sum", []byte(in))
	if err != nil {
		t.Fatal(err)
	}

	// Drop without cleanup - FormatSum should still skip cleared entries.
	mod := module.Version{Path: "golang.org/x/text", Version: "v0.3.0"}
	f.DropHash(mod, false)

	got := string(FormatSum(f))
	want := "rsc.io/quote v1.5.2 h1:ghi789=\n"
	if got != want {
		t.Errorf("FormatSum without Cleanup:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSumAddThenFormat(t *testing.T) {
	f := &SumFile{}
	f.AddHash(module.Version{Path: "example.com/mod", Version: "v1.0.0"}, false, "h1:zip=")
	f.AddHash(module.Version{Path: "example.com/mod", Version: "v1.0.0"}, true, "h1:gomod=")

	got := string(FormatSum(f))
	want := "example.com/mod v1.0.0 h1:zip=\n" +
		"example.com/mod v1.0.0/go.mod h1:gomod=\n"
	if got != want {
		t.Errorf("Add then Format:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
