package wineregdiff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	fileHeader = "WINE REGISTRY Version 2"
)

var (
	errNotWineRegistryFile = errors.New("not a wine registry file (header not found)")
	keyPattern             = regexp.MustCompile(`\[(.+)]\s+(\d+)`)
	valuePattern           = regexp.MustCompile(`(".+"|@)=(.+)`)
	archPattern            = regexp.MustCompile(`^#arch=(.+)`)
	timePattern            = regexp.MustCompile(`^#time=([0-9a-fA-F]+)`)
	classPattern           = regexp.MustCompile(`^#class="(.+)"`)
)

func Parse(r io.Reader) (Registry, error) {
	rf, err := ParseFile(r)
	if err != nil {
		return nil, err
	}
	return rf.Registry, nil
}

// ParseFile parses a Wine registry file and returns a RegistryFile with metadata
func ParseFile(r io.Reader) (*RegistryFile, error) {
	scanner, err := newWineRegScannerWithMetadata(r)
	if err != nil {
		return nil, err
	}

	rf := NewRegistryFile()
	rf.FileMetadata.Comment = scanner.comment
	rf.FileMetadata.Arch = scanner.arch

	var subKey *Key
	var currentTimestamp int64

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "["):
			matches := keyPattern.FindStringSubmatch(line)
			if len(matches) < 3 {
				continue
			}
			key := Key(parseQuotedString(matches[1]))
			timestamp := matches[2]
			subKey = &key
			if _, ok := rf.Registry[*subKey]; !ok {
				rf.Registry[*subKey] = Value{}
			}
			// Parse timestamp
			var ts int64
			fmt.Sscanf(timestamp, "%d", &ts)
			currentTimestamp = ts
		case strings.HasPrefix(line, "#time="):
			if subKey != nil {
				matches := timePattern.FindStringSubmatch(line)
				if len(matches) >= 2 {
					meta := rf.KeyMetadata[*subKey]
					meta.Timestamp = currentTimestamp
					meta.Time = matches[1]
					rf.KeyMetadata[*subKey] = meta
				}
			}
		// Key-level metadata: #class and #link
		// https://github.com/wine-mirror/wine/blob/614887f33c5a0269d5e2feff6891e3e90d0f77ee/server/registry.c (load_key_option)
		case strings.HasPrefix(line, "#class="):
			if subKey != nil {
				matches := classPattern.FindStringSubmatch(line)
				if len(matches) >= 2 {
					meta := rf.KeyMetadata[*subKey]
					meta.Class = matches[1]
					rf.KeyMetadata[*subKey] = meta
				}
			}
		case line == "#link":
			if subKey != nil {
				meta := rf.KeyMetadata[*subKey]
				meta.Link = true
				rf.KeyMetadata[*subKey] = meta
			}
		case strings.HasPrefix(line, `"`) || strings.HasPrefix(line, string(UnnamedDataName)):
			if subKey == nil {
				return nil, errors.New("invalid value (no subkey)")
			}
			matches := valuePattern.FindStringSubmatch(line)
			if len(matches) < 3 {
				continue
			}
			dataName, val := DataName(parseQuotedString(matches[1])), matches[2]
			data, err := ParseData(val)
			if err != nil {
				return nil, fmt.Errorf("failed to parse data(key: %s, name: %s): %+v", *subKey, dataName, err)
			}
			rf.Registry[*subKey][dataName] = data
		default:
			// ignore other lines
		}
	}
	return rf, nil
}

type wineRegScanner struct {
	sc      *bufio.Scanner
	line    string
	comment string
	arch    string
}

func newWineRegScanner(r io.Reader) (*wineRegScanner, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return nil, errNotWineRegistryFile
	}
	if sc.Text() != fileHeader {
		return nil, errNotWineRegistryFile
	}
	return &wineRegScanner{
		sc: sc,
	}, nil
}

func newWineRegScannerWithMetadata(r io.Reader) (*wineRegScanner, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		return nil, errNotWineRegistryFile
	}
	if sc.Text() != fileHeader {
		return nil, errNotWineRegistryFile
	}

	scanner := &wineRegScanner{sc: sc}

	// Read metadata lines until we hit a key or EOF
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ";;") {
			scanner.comment = strings.TrimPrefix(line, ";; ")
			continue
		}
		if matches := archPattern.FindStringSubmatch(line); len(matches) >= 2 {
			scanner.arch = matches[1]
			continue
		}
		// If we hit a key, put it back by storing in line
		if strings.HasPrefix(line, "[") {
			scanner.line = line
			return scanner, nil
		}
	}
	return scanner, nil
}

func (s *wineRegScanner) Scan() bool {
	// If line is already set (from metadata scanning), return it first
	if s.line != "" {
		return true
	}
	if !s.sc.Scan() {
		return false
	}
	s.line = strings.TrimSpace(s.sc.Text())
	if strings.HasSuffix(s.line, `\`) {
		s.line = strings.TrimSuffix(s.line, `\`)
		for {
			if !s.sc.Scan() {
				return false
			}
			add := strings.TrimSpace(s.sc.Text())
			cont := strings.HasSuffix(add, `\`)
			if cont {
				add = strings.TrimSuffix(add, `\`)
			}
			s.line += add
			if !cont {
				break
			}
		}
	}
	return true
}

func (s *wineRegScanner) Text() string {
	line := s.line
	s.line = "" // Clear line so next Scan() reads from scanner
	return line
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// parseQuotedString unescapes a Wine registry quoted string.
// Wine uses C-style escapes: \\, \", \a, \b, \e, \f, \n, \r, \t, \v,
// \xNNNN (1-4 hex digits for UTF-16 code unit), and \NNN (1-3 octal digits).
// https://github.com/wine-mirror/wine/blob/614887f33c5a0269d5e2feff6891e3e90d0f77ee/server/unicode.c (parse_strW)
func parseQuotedString(s string) string {
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			buf.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case '\\':
			buf.WriteByte('\\')
		case '"':
			buf.WriteByte('"')
		case 'a':
			buf.WriteByte('\a')
		case 'b':
			buf.WriteByte('\b')
		case 'e':
			buf.WriteByte(0x1b)
		case 'f':
			buf.WriteByte('\f')
		case 'n':
			buf.WriteByte('\n')
		case 'r':
			buf.WriteByte('\r')
		case 't':
			buf.WriteByte('\t')
		case 'v':
			buf.WriteByte('\v')
		case 'x':
			// \xNNNN: 1-4 hex digits
			var val rune
			digits := 0
			for digits < 4 && i+1 < len(s) && isHexDigit(s[i+1]) {
				val = val*16 + rune(hexVal(s[i+1]))
				i++
				digits++
			}
			if digits > 0 {
				buf.WriteRune(val)
			} else {
				buf.WriteByte('x')
			}
		case '0', '1', '2', '3', '4', '5', '6', '7':
			// \NNN: 1-3 octal digits
			val := rune(s[i] - '0')
			for digits := 1; digits < 3 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7'; digits++ {
				i++
				val = val*8 + rune(s[i]-'0')
			}
			buf.WriteRune(val)
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}
