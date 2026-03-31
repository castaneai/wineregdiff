package wineregdiff

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const wineFileHeader = "WINE REGISTRY Version 2"

// WriteWineFormat writes the Registry to io.Writer in Wine native format
func WriteWineFormat(w io.Writer, reg Registry) error {
	rf := &RegistryFile{
		Registry:    reg,
		KeyMetadata: make(map[Key]KeyMetadata),
	}
	return WriteRegistryFile(w, rf)
}

// WriteRegistryFile writes the RegistryFile to io.Writer in Wine native format with metadata
func WriteRegistryFile(w io.Writer, rf *RegistryFile) error {
	// Write header
	if _, err := fmt.Fprintln(w, wineFileHeader); err != nil {
		return err
	}

	// Write file-level metadata
	if rf.FileMetadata.Comment != "" {
		if _, err := fmt.Fprintf(w, ";; %s\n", rf.FileMetadata.Comment); err != nil {
			return err
		}
	}
	if rf.FileMetadata.Arch != "" {
		if _, err := fmt.Fprintf(w, "\n#arch=%s\n", rf.FileMetadata.Arch); err != nil {
			return err
		}
	}

	// Sort keys for consistent output
	keys := make([]Key, 0, len(rf.Registry))
	for key := range rf.Registry {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	timestamp := time.Now().Unix()

	for _, key := range keys {
		value := rf.Registry[key]
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}

		// Write key-level metadata in the same order as Wine's save_subkeys():
		// https://github.com/wine-mirror/wine/blob/614887f33c5a0269d5e2feff6891e3e90d0f77ee/server/registry.c (save_subkeys)
		keyTimestamp := timestamp
		escapedKey := escapeWineKey(string(key))
		if meta, ok := rf.KeyMetadata[key]; ok {
			if meta.Timestamp != 0 {
				keyTimestamp = meta.Timestamp
			}
			if _, err := fmt.Fprintf(w, "[%s] %d\n", escapedKey, keyTimestamp); err != nil {
				return err
			}
			if meta.Time != "" {
				if _, err := fmt.Fprintf(w, "#time=%s\n", meta.Time); err != nil {
					return err
				}
			}
			if meta.Class != "" {
				if _, err := fmt.Fprintf(w, "#class=\"%s\"\n", escapeWineString(meta.Class)); err != nil {
					return err
				}
			}
			if meta.Link {
				if _, err := fmt.Fprintln(w, "#link"); err != nil {
					return err
				}
			}
		} else {
			if _, err := fmt.Fprintf(w, "[%s] %d\n", escapedKey, keyTimestamp); err != nil {
				return err
			}
		}

		// Write values
		if err := writeWineValues(w, value); err != nil {
			return err
		}
	}
	return nil
}

func escapeWineKey(key string) string {
	// Wine format escapes backslashes
	return strings.ReplaceAll(key, `\`, `\\`)
}

func writeWineValues(w io.Writer, value Value) error {
	// Sort value names
	dataNames := make([]DataName, 0, len(value))
	for name := range value {
		dataNames = append(dataNames, name)
	}
	sort.Slice(dataNames, func(i, j int) bool { return dataNames[i] < dataNames[j] })

	for _, dataName := range dataNames {
		data := value[dataName]
		wineValue := dataToWineString(data)
		if dataName == UnnamedDataName {
			if _, err := fmt.Fprintf(w, "@=%s\n", wineValue); err != nil {
				return err
			}
		} else {
			escapedName := escapeWineString(string(dataName))
			if _, err := fmt.Fprintf(w, "\"%s\"=%s\n", escapedName, wineValue); err != nil {
				return err
			}
		}
	}
	return nil
}

// dataToWineString converts Data to Wine native format string
func dataToWineString(data Data) string {
	switch d := data.(type) {
	case StringData:
		// Wine format: "value" (escaped)
		return fmt.Sprintf(`"%s"`, escapeWineString(string(d)))
	case ExpandStringData:
		// Wine format: str(2):"value"
		return fmt.Sprintf(`str(2):"%s"`, escapeWineString(string(d)))
	case MultiStringData:
		// Wine format: str(7):"value1\0value2\0"
		// Escape each string then join with \0
		var escaped []string
		for _, s := range d {
			escaped = append(escaped, escapeWineString(s))
		}
		joined := strings.Join(escaped, `\0`)
		return fmt.Sprintf(`str(7):"%s"`, joined)
	default:
		// DwordData, BinaryData, UnknownData can use String() as-is
		return data.String()
	}
}

// escapeWineString escapes a string for Wine registry format.
// Wine uses \\ for backslash, \" for quotes, C-style escapes for control chars,
// and \xNNNN for non-ASCII characters (UTF-16 code units).
// https://github.com/wine-mirror/wine/blob/614887f33c5a0269d5e2feff6891e3e90d0f77ee/server/unicode.c (dump_strW)
func escapeWineString(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\\':
			buf.WriteString(`\\`)
		case r == '"':
			buf.WriteString(`\"`)
		case r == '\a':
			buf.WriteString(`\a`)
		case r == '\b':
			buf.WriteString(`\b`)
		case r == 0x1b:
			buf.WriteString(`\e`)
		case r == '\f':
			buf.WriteString(`\f`)
		case r == '\n':
			buf.WriteString(`\n`)
		case r == '\r':
			buf.WriteString(`\r`)
		case r == '\t':
			buf.WriteString(`\t`)
		case r == '\v':
			buf.WriteString(`\v`)
		case r < 32:
			fmt.Fprintf(&buf, `\x%04x`, r)
		case r > 127:
			if r > 0xFFFF {
				// Surrogate pair
				r -= 0x10000
				high := 0xD800 + (r >> 10)
				low := 0xDC00 + (r & 0x3FF)
				fmt.Fprintf(&buf, `\x%04x\x%04x`, high, low)
			} else {
				fmt.Fprintf(&buf, `\x%04x`, r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
