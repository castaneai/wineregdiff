package wineregdiff

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
)

type Registry map[Key]Value
type Key string
type Value map[DataName]Data
type DataName string
type Data interface{}
type DataType int

const (
	fileHeader = "WINE REGISTRY Version 2"

	// https://github.com/wine-mirror/wine/blob/e909986e6ea5ecd49b2b847f321ad89b2ae4f6f1/include/winnt.h#L5571
	DataTypeRegSZ             DataType = 1
	DataTypeRegExpandSZ       DataType = 2
	DataTypeRegBinary         DataType = 3
	DataTypeRegDWord          DataType = 4
	DataTypeRegDwordBigEndian DataType = 5
	DataTypeRegLink           DataType = 6
	DataTypeRegMultiSZ        DataType = 7
)

var (
	errNotWineRegistryFile = errors.New("not a wine registry file (header not found)")
	keyPattern = regexp.MustCompile(`\[(.+)]\s+(\d+)`)
	valuePattern = regexp.MustCompile(`(".+"|@)=(.+)`)
)

func Parse(r io.Reader) (Registry, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return nil, errNotWineRegistryFile
	}
	if scanner.Text() != fileHeader {
		return nil, errNotWineRegistryFile
	}

	reg := Registry{}
	var subKey *Key

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "["):
			matches := keyPattern.FindStringSubmatch(line)
			key, _ := Key(matches[1]), matches[2]
			subKey = &key
		case strings.HasPrefix(line, `"`):
			if subKey == nil {
				return nil, errors.New("invalid value (no subkey)")
			}
			matches := valuePattern.FindStringSubmatch(line)
			dataName, val := DataName(matches[0]), matches[1]
			if _, ok := reg[*subKey]; !ok {
				reg[*subKey] = Value{}
			}
			reg[*subKey][dataName] = val
		case strings.HasPrefix(line, ";"): // comment
		}
	}
	return reg, nil
}

type RegistryDiff struct {
	Registry1Only Registry
	Registry2Only Registry
	RegistryChanged map[Key]ValueDiff
}

func NewRegistryDiff() RegistryDiff {
	return RegistryDiff{
		Registry1Only:   Registry{},
		Registry2Only:   Registry{},
		RegistryChanged: map[Key]ValueDiff{},
	}
}

type ValueDiff struct {
	Key     Key
	Value1 Value
	Value2 Value
}

func (d ValueDiff) HasDiff() bool {
	return len(d.Value1) > 0 || len(d.Value2) > 0
}

func NewValueDiff(key Key) ValueDiff {
	return ValueDiff{
		Key:    key,
		Value1: Value{},
		Value2: Value{},
	}
}

func Diff(reg1, reg2 Registry) (RegistryDiff, error) {
	cmp := &DefaultValueComparator{
		DataComparator: &DefaultDataComparator{},
	}
	diff := NewRegistryDiff()
	for key, value1 := range reg1 {
		value2, ok := reg2[key]
		if !ok {
			diff.Registry1Only[key] = value1
			continue
		}
		valueDiff, err := cmp.CompareValue(key, value1, value2)
		if err != nil {
			return diff, err
		}
		if valueDiff.HasDiff() {
			diff.RegistryChanged[key] = valueDiff
		}
	}
	for key, value2 := range reg2 {
		if _, ok := reg1[key]; !ok {
			diff.Registry2Only[key] = value2
		}
	}
	return diff, nil
}