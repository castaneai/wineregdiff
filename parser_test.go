package wineregdiff

import (
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"path"
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home dir: %+v", err)
	}
	regPath :=  path.Join(homeDir, ".wine", "user.reg")
	newTestReg(t, regPath)
}

func TestDiff(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home dir: %+v", err)
	}
	regPath :=  path.Join(homeDir, ".wine", "user.reg")
	reg1 := newTestReg(t, regPath)
	reg2 := newTestReg(t, regPath)

	diff, err := Diff(reg1, reg2)
	assert.NoError(t, err)

	log.Printf("--- reg1 only ---")
	for key, value := range diff.Registry1Only {
		log.Printf("  [%s]", key)
		for name, data := range value {
			log.Printf("  %s=%s", name, data)
		}
	}
	log.Printf("--- reg2 only ---")
	for key, value := range diff.Registry2Only {
		log.Printf("  [%s]", key)
		for name, data := range value {
			log.Printf("  %s=%s", name, data)
		}
	}
	log.Printf("--- changed ---")
	for key, valueDiff := range diff.RegistryChanged {
		log.Printf("  [%s]", key)
		log.Printf("  --- reg1 ---")
		for name, data := range valueDiff.Value1 {
			log.Printf("  %s=%s", name, data)
		}
		log.Printf("  --- reg2 ---")
		for name, data := range valueDiff.Value2 {
			log.Printf("  %s=%s", name, data)
		}
	}
}