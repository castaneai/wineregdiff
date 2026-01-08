package wineregdiff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_Merge(t *testing.T) {
	base := Registry{
		Key("Software\\Test"): Value{
			DataName("existing"): StringData("old value"),
			DataName("keep"):     StringData("keep this"),
		},
	}

	other := Registry{
		Key("Software\\Test"): Value{
			DataName("existing"): StringData("new value"),
			DataName("added"):    StringData("added value"),
		},
		Key("Software\\New"): Value{
			DataName("newkey"): DwordData(123),
		},
	}

	base.Merge(other)

	// Existing value in existing key is overwritten
	assert.Equal(t, StringData("new value"), base[Key("Software\\Test")][DataName("existing")])
	// Existing value in existing key is preserved
	assert.Equal(t, StringData("keep this"), base[Key("Software\\Test")][DataName("keep")])
	// New value is added to existing key
	assert.Equal(t, StringData("added value"), base[Key("Software\\Test")][DataName("added")])
	// New key is added
	assert.Equal(t, DwordData(123), base[Key("Software\\New")][DataName("newkey")])
}

func TestRegistry_Clone(t *testing.T) {
	original := Registry{
		Key("Software\\Test"): Value{
			DataName("value"): StringData("test"),
		},
	}

	clone := original.Clone()

	// Clone has the same values as original
	assert.Equal(t, original, clone)

	// Modifying clone does not affect original
	clone[Key("Software\\Test")][DataName("value")] = StringData("modified")
	assert.Equal(t, StringData("test"), original[Key("Software\\Test")][DataName("value")])
}
