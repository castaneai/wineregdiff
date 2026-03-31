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

func TestWriteRegistryFile_LinkAndClass(t *testing.T) {
	rf := &RegistryFile{
		FileMetadata: FileMetadata{Arch: "win64"},
		Registry: Registry{
			Key("Software\\Classes\\Wow6432Node"): Value{
				DataName("value"): StringData("test"),
			},
			Key("Software\\Normal"): Value{
				DataName("value"): StringData("hello"),
			},
		},
		KeyMetadata: map[Key]KeyMetadata{
			Key("Software\\Classes\\Wow6432Node"): {
				Timestamp: 1628790650,
				Time:      "1dc6e34610bc106",
				Class:     "Wow64Class",
				Link:      true,
			},
			Key("Software\\Normal"): {
				Timestamp: 1628790650,
				Time:      "1dc6e34610bc106",
			},
		},
	}

	var buf bytes.Buffer
	err := WriteRegistryFile(&buf, rf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#class=\"Wow64Class\"\n")
	assert.Contains(t, output, "#link\n")

	// Ensure #link does not appear for non-link keys
	// Split by key headers and check the "Normal" section
	sections := strings.Split(output, "\n\n")
	for _, section := range sections {
		if strings.Contains(section, "Normal") {
			assert.NotContains(t, section, "#link")
			assert.NotContains(t, section, "#class")
		}
	}
}

func TestWriteRegistryFile_LinkAndClass_RoundTrip(t *testing.T) {
	rf := &RegistryFile{
		Registry: Registry{
			Key("Software\\Link"): Value{
				DataName("value"): StringData("test"),
			},
		},
		KeyMetadata: map[Key]KeyMetadata{
			Key("Software\\Link"): {
				Timestamp: 1628790650,
				Time:      "1dc6e34610bc106",
				Class:     "MyClass",
				Link:      true,
			},
		},
	}

	var buf bytes.Buffer
	err := WriteRegistryFile(&buf, rf)
	require.NoError(t, err)

	parsed, err := ParseFile(&buf)
	require.NoError(t, err)

	meta := parsed.KeyMetadata[Key("Software\\Link")]
	assert.Equal(t, "MyClass", meta.Class)
	assert.True(t, meta.Link)
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
