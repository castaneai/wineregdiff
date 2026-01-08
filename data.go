package wineregdiff

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type DataType int

const (
	// https://github.com/wine-mirror/wine/blob/e909986e6ea5ecd49b2b847f321ad89b2ae4f6f1/include/winnt.h#L5571
	DataTypeRegNone           DataType = 0
	DataTypeRegSZ             DataType = 1
	DataTypeRegExpandSZ       DataType = 2
	DataTypeRegBinary         DataType = 3
	DataTypeRegDWord          DataType = 4
	DataTypeRegDwordBigEndian DataType = 5
	DataTypeRegLink           DataType = 6
	DataTypeRegMultiSZ        DataType = 7
)

func (t DataType) String() string {
	switch t {
	case DataTypeRegSZ:
		return "REG_SZ"
	case DataTypeRegExpandSZ:
		return "REG_EXPAND_SZ"
	case DataTypeRegBinary:
		return "REG_BINARY"
	case DataTypeRegDWord:
		return "REG_DWORD"
	case DataTypeRegMultiSZ:
		return "REG_MULTI_SZ"
	default:
		return "REG_NONE"
	}
}

type Data interface {
	fmt.Stringer
	DataType() DataType
	CommandString() string
	RegFileString() string
}

var (
	stringTagPattern      = regexp.MustCompile(`^str\(([0-9a-fA-F]+)\):(.+)`)
	unknownDataTagPattern = regexp.MustCompile(`^hex\(([0-9a-fA-F]+)\):(.*)`)
)

type StringData string

func (d StringData) DataType() DataType {
	return DataTypeRegSZ
}

func (d StringData) String() string {
	return string(d)
}

func (d StringData) CommandString() string {
	return escapeString(string(d))
}

func (d StringData) RegFileString() string {
	return fmt.Sprintf(`"%s"`, escapeRegFileString(string(d)))
}

type ExpandStringData string

func (d ExpandStringData) DataType() DataType {
	return DataTypeRegExpandSZ
}

func (d ExpandStringData) String() string {
	return string(d)
}

func (d ExpandStringData) CommandString() string {
	return escapeString(string(d))
}

func (d ExpandStringData) RegFileString() string {
	return fmt.Sprintf("hex(2):%s", toUTF16LEHex(string(d)))
}

type MultiStringData []string

func (d MultiStringData) DataType() DataType {
	return DataTypeRegMultiSZ
}

func (d MultiStringData) String() string {
	return strings.Join(d, `\0`)
}

func (d MultiStringData) CommandString() string {
	return escapeString(d.String())
}

func (d MultiStringData) RegFileString() string {
	return fmt.Sprintf("hex(7):%s", toUTF16LEHexMulti(d))
}

type DwordData uint32

func (d DwordData) DataType() DataType {
	return DataTypeRegDWord
}

func (d DwordData) String() string {
	return fmt.Sprintf("dword:%08x", uint32(d))
}

func (d DwordData) CommandString() string {
	return fmt.Sprintf("%d", uint32(d))
}

func (d DwordData) RegFileString() string {
	return fmt.Sprintf("dword:%08x", uint32(d))
}

type BinaryData []byte

func (d BinaryData) DataType() DataType {
	return DataTypeRegBinary
}

func (d BinaryData) String() string {
	return fmt.Sprintf("hex:%s", asHex(d))
}

func (d BinaryData) CommandString() string {
	return hex.EncodeToString(d)
}

func (d BinaryData) RegFileString() string {
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
	return fmt.Sprintf("hex(%x):%s", int(d.DataType()), asHex(d.Data))
}

func (d *UnknownData) CommandString() string {
	return hex.EncodeToString(d.Data)
}

func (d *UnknownData) RegFileString() string {
	return fmt.Sprintf("hex(%x):%s", int(d.DataType()), asHex(d.Data))
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
	matches := stringTagPattern.FindStringSubmatch(s)
	if len(matches) > 2 {
		dt, err := strconv.ParseUint(matches[1], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse unknown type as hex('%s'): %+v", s, err)
		}
		dataType := DataType(dt)
		switch dataType {
		case DataTypeRegExpandSZ:
			data := parseQuotedString(matches[2])
			return ExpandStringData(data), nil
		case DataTypeRegMultiSZ:
			data := strings.Split(parseQuotedString(matches[2]), `\0`)
			return MultiStringData(data), nil
		}
	}
	matches = unknownDataTagPattern.FindStringSubmatch(s)
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
	if s == "" {
		return []byte{}, nil
	}
	digits := strings.Split(s, ",")
	var data []byte
	for _, d := range digits {
		hexd, err := strconv.ParseUint(d, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to parse as binary('%s'): %+v", s, err)
		}
		data = append(data, byte(hexd))
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

func escapeRegFileString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func toUTF16LEHex(s string) string {
	var data []byte
	for _, r := range s {
		// UTF-16LE encoding
		if r <= 0xFFFF {
			data = append(data, byte(r), byte(r>>8))
		} else {
			// Surrogate pair for characters > 0xFFFF
			r -= 0x10000
			high := 0xD800 + (r >> 10)
			low := 0xDC00 + (r & 0x3FF)
			data = append(data, byte(high), byte(high>>8))
			data = append(data, byte(low), byte(low>>8))
		}
	}
	// Null terminator
	data = append(data, 0x00, 0x00)
	return asHex(data)
}

func toUTF16LEHexMulti(ss []string) string {
	var data []byte
	for _, s := range ss {
		for _, r := range s {
			if r <= 0xFFFF {
				data = append(data, byte(r), byte(r>>8))
			} else {
				r -= 0x10000
				high := 0xD800 + (r >> 10)
				low := 0xDC00 + (r & 0x3FF)
				data = append(data, byte(high), byte(high>>8))
				data = append(data, byte(low), byte(low>>8))
			}
		}
		// Null terminator for each string
		data = append(data, 0x00, 0x00)
	}
	// Double null terminator at the end
	data = append(data, 0x00, 0x00)
	return asHex(data)
}
