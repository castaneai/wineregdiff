package wineregdiff

import (
	"os"
	"strings"
	"testing"
)

func newTestReg(t *testing.T, filename string) Registry {
	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open %s: %+v", filename, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	reg, err := Parse(f)
	if err != nil {
		t.Fatalf("failed to parse reg file %s: %+v", filename, err)
	}
	return reg
}

func TestParse(t *testing.T) {
	newTestReg(t, "testdata/example1.reg")
	newTestReg(t, "testdata/example2.reg")
}

func TestParseFile_LinkAndClass(t *testing.T) {
	input := `WINE REGISTRY Version 2

[Software\\Classes\\Wow6432Node] 1628790650
#time=1dc6e34610bc106
#class="Wow64Class"
#link
"SymbolicLinkValue"="target"

[Software\\Normal] 1628790650
#time=1dc6e34610bc106
"value"="hello"

[Software\\LinkOnly] 1628790650
#link
`
	rf, err := ParseFile(strings.NewReader(input))
	if err != nil {
		t.Fatalf("failed to parse: %+v", err)
	}

	// Key with both #class and #link
	classAndLinkMeta := rf.KeyMetadata[Key(`Software\Classes\Wow6432Node`)]
	if classAndLinkMeta.Class != "Wow64Class" {
		t.Errorf("expected Class=Wow64Class, got %q", classAndLinkMeta.Class)
	}
	if !classAndLinkMeta.Link {
		t.Error("expected Link=true for Wow6432Node")
	}

	// Key without #link or #class
	normalMeta := rf.KeyMetadata[Key(`Software\Normal`)]
	if normalMeta.Class != "" {
		t.Errorf("expected empty Class, got %q", normalMeta.Class)
	}
	if normalMeta.Link {
		t.Error("expected Link=false for Normal")
	}

	// Key with #link only (no #time, no #class)
	linkOnlyMeta := rf.KeyMetadata[Key(`Software\LinkOnly`)]
	if !linkOnlyMeta.Link {
		t.Error("expected Link=true for LinkOnly")
	}
}

func TestParseQuotedString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`"hello \"world\""`, `hello "world"`},
		{`"path\\to\\file"`, `path\to\file`},
		// \x hex escape
		{`"\x0041"`, "A"},
		{`"\x00e9"`, "\u00e9"},        // é
		{`"\x3042"`, "\u3042"},        // あ
		{`"test\x0041end"`, "testAend"},
		// C-style control char escapes
		{`"line1\nline2"`, "line1\nline2"},
		{`"col1\tcol2"`, "col1\tcol2"},
		{`"\a\b\f\r\v"`, "\a\b\f\r\v"},
		// octal escape
		{`"\101"`, "A"}, // 0101 = 65 = 'A'
	}
	for _, tt := range tests {
		got := parseQuotedString(tt.input)
		if got != tt.want {
			t.Errorf("parseQuotedString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseData_ToLowerPreservesStringValues(t *testing.T) {
	// str(2) values should preserve case
	data, err := ParseData(`str(2):"C:\\Users\\Foo"`)
	if err != nil {
		t.Fatalf("failed to parse: %+v", err)
	}
	expand, ok := data.(ExpandStringData)
	if !ok {
		t.Fatalf("expected ExpandStringData, got %T", data)
	}
	if string(expand) != `C:\Users\Foo` {
		t.Errorf("expected C:\\Users\\Foo, got %q", string(expand))
	}

	// str(7) values should preserve case
	data2, err := ParseData(`str(7):"Hello\0World"`)
	if err != nil {
		t.Fatalf("failed to parse: %+v", err)
	}
	multi, ok := data2.(MultiStringData)
	if !ok {
		t.Fatalf("expected MultiStringData, got %T", data2)
	}
	if multi[0] != "Hello" || multi[1] != "World" {
		t.Errorf("expected [Hello, World], got %v", multi)
	}
}
