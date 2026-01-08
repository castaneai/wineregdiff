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

		// Use metadata timestamp if available, otherwise use current time
		keyTimestamp := timestamp
		keyTime := ""
		if meta, ok := rf.KeyMetadata[key]; ok {
			if meta.Timestamp != 0 {
				keyTimestamp = meta.Timestamp
			}
			keyTime = meta.Time
		}

		// Write key (escaped path + timestamp)
		escapedKey := escapeWineKey(string(key))
		if _, err := fmt.Fprintf(w, "[%s] %d\n", escapedKey, keyTimestamp); err != nil {
			return err
		}

		// Write #time if available
		if keyTime != "" {
			if _, err := fmt.Fprintf(w, "#time=%s\n", keyTime); err != nil {
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

func escapeWineString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
