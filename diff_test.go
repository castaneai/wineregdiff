package wineregdiff

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	reg1 := newTestReg(t, "testdata/example1.reg")
	reg2 := newTestReg(t, "testdata/example2.reg")

	diff, err := Diff(reg1, reg2)
	assert.NoError(t, err)

	log.Printf("--- reg1 only ---")
	assert.Equal(t, 2, len(diff.Registry1Only))
	assert.Equal(t, `hello "world"`, diff.Registry1Only[`Example1 Only Key`]["StringValue"].(StringData).String())
	assert.Equal(t, `hello "world"`, diff.Registry1Only["Parent Key\\Example1 Only Sub Key"]["StringValue"].(StringData).String())
	assert.Equal(t, 2, len(diff.Registry2Only))
	assert.Equal(t, `hello "world2"`, diff.Registry2Only[`Example2 Only Key`]["StringValue"].(StringData).String())
	assert.Equal(t, `hello "world2"`, diff.Registry2Only["Parent Key\\Example2 Only Sub Key"]["StringValue"].(StringData).String())
	assert.Equal(t, 1, len(diff.RegistryChanged))
	assert.Equal(t, 2, len(diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value1))
	assert.Equal(t, 2, len(diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value2))
	assert.Equal(t, `hello "world"`, diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value1["StringValue"].(StringData).String())
	assert.Equal(t, `hello "world2"`, diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value2["StringValue"].(StringData).String())
	assert.Equal(t, []byte{0xab, 0xcd, 0xef}, diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value1["UnknownValue"].(*UnknownData).Data)
	assert.Equal(t, []byte{0xab, 0xcd, 0xef, 0x12}, diff.RegistryChanged["Parent Key\\Changed Sub Key"].Value2["UnknownValue"].(*UnknownData).Data)
}
