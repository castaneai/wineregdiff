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
)

func Parse(r io.Reader) (Registry, error) {
	scanner, err := newWineRegScanner(r)
	if err != nil {
		return nil, err
	}
	reg := Registry{}
	var subKey *Key

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "["):
			matches := keyPattern.FindStringSubmatch(line)
			key, _ := Key(parseQuotedString(matches[1])), matches[2]
			subKey = &key
			if _, ok := reg[*subKey]; !ok {
				reg[*subKey] = Value{}
			}
		case strings.HasPrefix(line, `"`) || strings.HasPrefix(line, string(UnnamedDataName)):
			if subKey == nil {
				return nil, errors.New("invalid value (no subkey)")
			}
			matches := valuePattern.FindStringSubmatch(line)
			dataName, val := DataName(parseQuotedString(matches[1])), matches[2]
			data, err := ParseData(val)
			if err != nil {
				return nil, fmt.Errorf("failed to parse data(key: %s, name: %s): %+v", *subKey, dataName, err)
			}
			reg[*subKey][dataName] = data
		case strings.HasPrefix(line, ";"): // comment
		}
	}
	return reg, nil
}

type wineRegScanner struct {
	sc   *bufio.Scanner
	line string
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

func (s *wineRegScanner) Scan() bool {
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
	return s.line
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
