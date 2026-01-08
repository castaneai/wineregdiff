package wineregdiff

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRoundTrip_SystemReg(t *testing.T) {
	// Parse old_system.reg with metadata
	oldFile, err := os.Open("old_system.reg")
	require.NoError(t, err)
	defer oldFile.Close()

	oldRF, err := ParseFile(oldFile)
	require.NoError(t, err)

	// Parse system.reg with metadata
	sysFile, err := os.Open("system.reg")
	require.NoError(t, err)
	defer sysFile.Close()

	sysRF, err := ParseFile(sysFile)
	require.NoError(t, err)

	// Compute diff between old and new
	diff, err := Diff(oldRF.Registry, sysRF.Registry)
	require.NoError(t, err)

	// Clone old registry file and apply diff
	merged := oldRF.Clone()

	// Add entries that are only in system.reg
	for key, value := range diff.Registry2Only {
		merged.Registry[key] = value
		// Copy metadata if available
		if meta, ok := sysRF.KeyMetadata[key]; ok {
			merged.KeyMetadata[key] = meta
		}
	}

	// Update changed entries
	for key, changed := range diff.RegistryChanged {
		if _, ok := merged.Registry[key]; !ok {
			merged.Registry[key] = make(Value)
		}
		// Remove old values (Value1)
		for dataName := range changed.Value1 {
			delete(merged.Registry[key], dataName)
		}
		// Add new values (Value2)
		for dataName, data := range changed.Value2 {
			merged.Registry[key][dataName] = data
		}
		// Update metadata from sysRF
		if meta, ok := sysRF.KeyMetadata[key]; ok {
			merged.KeyMetadata[key] = meta
		}
	}

	// Delete entries that are only in old_system.reg
	for key, value := range diff.Registry1Only {
		if len(value) == 0 {
			// Delete entire key
			delete(merged.Registry, key)
			delete(merged.KeyMetadata, key)
		} else {
			// Delete specific values
			if existingValue, ok := merged.Registry[key]; ok {
				for dataName := range value {
					delete(existingValue, dataName)
				}
				if len(existingValue) == 0 {
					delete(merged.Registry, key)
					delete(merged.KeyMetadata, key)
				}
			}
		}
	}

	// Sync all metadata from sysRF (including unchanged keys)
	// This is needed because timestamps may be updated even when values are unchanged
	for key := range merged.Registry {
		if meta, ok := sysRF.KeyMetadata[key]; ok {
			merged.KeyMetadata[key] = meta
		}
	}

	// Update file-level metadata from sysRF
	merged.FileMetadata = sysRF.FileMetadata

	// Write merged registry to buffer
	var buf bytes.Buffer
	err = WriteRegistryFile(&buf, merged)
	require.NoError(t, err)

	// Save content before parsing consumes the buffer
	mergedContent := buf.String()

	// Write to new_system.reg for visual inspection
	err = os.WriteFile("new_system.reg", []byte(mergedContent), 0644)
	require.NoError(t, err)
	t.Log("Wrote merged result to new_system.reg")

	// Parse the written content back
	newRF, err := ParseFile(bytes.NewReader([]byte(mergedContent)))
	require.NoError(t, err)

	// Compare with original system.reg
	finalDiff, err := Diff(newRF.Registry, sysRF.Registry)
	require.NoError(t, err)

	// There should be no differences
	assert.Empty(t, finalDiff.Registry1Only, "newReg has entries not in sysReg")
	assert.Empty(t, finalDiff.Registry2Only, "sysReg has entries not in newReg")
	assert.Empty(t, finalDiff.RegistryChanged, "values differ between newReg and sysReg")

	t.Logf("old_system.reg keys: %d", len(oldRF.Registry))
	t.Logf("system.reg keys: %d", len(sysRF.Registry))
	t.Logf("merged keys: %d", len(merged.Registry))
	t.Logf("newReg keys: %d", len(newRF.Registry))

	// Also compare file content: write system.reg and compare with merged output
	var sysRegBuf bytes.Buffer
	err = WriteRegistryFile(&sysRegBuf, sysRF)
	require.NoError(t, err)

	// Compare byte-by-byte
	sysRegContent := sysRegBuf.String()

	if mergedContent == sysRegContent {
		t.Log("File content is IDENTICAL")
	} else {
		t.Logf("merged content length: %d bytes", len(mergedContent))
		t.Logf("sysReg content length: %d bytes", len(sysRegContent))

		// Find first difference
		minLen := len(mergedContent)
		if len(sysRegContent) < minLen {
			minLen = len(sysRegContent)
		}
		for i := 0; i < minLen; i++ {
			if mergedContent[i] != sysRegContent[i] {
				start := i - 50
				if start < 0 {
					start = 0
				}
				end := i + 50
				if end > minLen {
					end = minLen
				}
				t.Logf("First difference at byte %d", i)
				t.Logf("merged around diff: %q", mergedContent[start:end])
				t.Logf("sysReg around diff: %q", sysRegContent[start:end])
				break
			}
		}
	}

	assert.Equal(t, sysRegContent, mergedContent, "file content should match")
}
