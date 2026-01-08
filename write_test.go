package wineregdiff

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteWineFormat(t *testing.T) {
	reg := Registry{
		Key("Software\\Test"): Value{
			DataName("string"):  StringData("hello"),
			DataName("dword"):   DwordData(42),
			DataName("binary"):  BinaryData{0x01, 0x02, 0x03},
			DataName("expand"):  ExpandStringData("%PATH%"),
			DataName("multi"):   MultiStringData{"one", "two"},
		},
	}

	var buf bytes.Buffer
	err := WriteWineFormat(&buf, reg)
	require.NoError(t, err)

	output := buf.String()

	// Header is included
	assert.True(t, strings.HasPrefix(output, "WINE REGISTRY Version 2\n"))

	// Key is included
	assert.Contains(t, output, "[Software\\\\Test]")

	// Values are included
	assert.Contains(t, output, `"string"="hello"`)
	assert.Contains(t, output, `"dword"=dword:0000002a`)
	assert.Contains(t, output, `"binary"=hex:01,02,03`)
	assert.Contains(t, output, `"expand"=str(2):"%PATH%"`)
	assert.Contains(t, output, `"multi"=str(7):"one\0two"`)
}

func TestWriteWineFormat_RoundTrip(t *testing.T) {
	// Original registry
	original := Registry{
		Key("Software\\Test\\SubKey"): Value{
			DataName("value"): StringData("test value"),
		},
	}

	// Write in Wine format
	var buf bytes.Buffer
	err := WriteWineFormat(&buf, original)
	require.NoError(t, err)

	// Parse back
	parsed, err := Parse(&buf)
	require.NoError(t, err)

	// Should have the same content as original
	assert.Equal(t, original[Key("Software\\Test\\SubKey")][DataName("value")], parsed[Key("Software\\Test\\SubKey")][DataName("value")])
}

func TestEscapeWineKey(t *testing.T) {
	// Backslashes are escaped
	assert.Equal(t, `Software\\Test`, escapeWineKey(`Software\Test`))
	assert.Equal(t, `Software\\Test\\SubKey`, escapeWineKey(`Software\Test\SubKey`))
}

func TestEscapeWineString(t *testing.T) {
	// Backslashes and double quotes are escaped
	assert.Equal(t, `path\\to\\file`, escapeWineString(`path\to\file`))
	assert.Equal(t, `say \"hello\"`, escapeWineString(`say "hello"`))
}
