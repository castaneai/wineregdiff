package wineregdiff

import (
	"os"
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
