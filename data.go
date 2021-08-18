package wineregdiff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// https://github.com/wine-mirror/wine/blob/e909986e6ea5ecd49b2b847f321ad89b2ae4f6f1/include/winnt.h#L5571
	DataTypeRegSZ             DataType = 1
	DataTypeRegExpandSZ       DataType = 2
	DataTypeRegBinary         DataType = 3
	DataTypeRegDWord          DataType = 4
	DataTypeRegDwordBigEndian DataType = 5
	DataTypeRegLink           DataType = 6
	DataTypeRegMultiSZ        DataType = 7
)

type DataType int
type Data interface {
	fmt.Stringer
	DataType() DataType
}

var (
	unknownDataTagPattern = regexp.MustCompile(`^hex\(([0-9a-fA-F]+)\):(.+)`)
)

type StringData string

func (d StringData) DataType() DataType {
	return DataTypeRegSZ
}
func (d StringData) String() string {
	return string(d)
}

type DwordData uint32

func (d DwordData) DataType() DataType {
	return DataTypeRegDWord
}
func (d DwordData) String() string {
	return fmt.Sprintf("dword:%08x", uint32(d))
}

type BinaryData []byte

func (d BinaryData) DataType() DataType {
	return DataTypeRegBinary
}
func (d BinaryData) String() string {
	return fmt.Sprintf("hex:%s", asHex(d))
}

// REG_NONE, REG_EXPAND_SZ, REG_MULTI_SZ, ...
// https://github.com/wine-mirror/wine/blob/60a3e0106246cb91d598a815d4fadf2791011142/programs/reg/export.c#L200-L204
type UnknownData struct {
	dataType DataType
	Data     []byte
}

func (d *UnknownData) DataType() DataType {
	return d.dataType
}
func (d *UnknownData) String() string {
	return fmt.Sprintf("hex(%x):%s", d.DataType(), asHex(d.Data))
}

// https://github.com/wine-mirror/wine/blob/60a3e0106246cb91d598a815d4fadf2791011142/programs/reg/import.c#L249
func ParseData(s string) (Data, error) {
	if strings.HasPrefix(s, `"`) {
		return StringData(parseQuotedString(s)), nil
	}
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "dword:") {
		d, err := strconv.ParseUint(strings.TrimPrefix(s, "dword:"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse as REG_DWORD('%s'): %+v", s, err)
		}
		return DwordData(d), nil
	}
	if strings.HasPrefix(s, "hex:") {
		data, err := parseHex(strings.TrimPrefix(s, "hex:"))
		if err != nil {
			return nil, err
		}
		return BinaryData(data), nil
	}
	matches := unknownDataTagPattern.FindStringSubmatch(s)
	if len(matches) > 2 {
		dataType, err := strconv.ParseUint(matches[1], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unknown type as hex('%s'): %+v", s, err)
		}
		data, err := parseHex(matches[2])
		if err != nil {
			return nil, err
		}
		return &UnknownData{Data: data, dataType: DataType(dataType)}, nil
	}
	return nil, fmt.Errorf("unknown input: '%s'", s)
}

func parseHex(s string) ([]byte, error) {
	digits := strings.Split(s, ",")
	var data []byte
	for _, d := range digits {
		hex, err := strconv.ParseUint(d, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to parse as binary('%s'): %+v", s, err)
		}
		data = append(data, byte(hex))
	}
	return data, nil
}

func asHex(data []byte) string {
	var ss []string
	for _, b := range data {
		ss = append(ss, fmt.Sprintf("%02x", b))
	}
	return strings.Join(ss, ",")
}
