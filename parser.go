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
					rf.KeyMetadata[*subKey] = KeyMetadata{
						Timestamp: currentTimestamp,
						Time:      matches[1],
					}
				}
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

func parseQuotedString(s string) string {
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
