package wineregdiff

import "fmt"

type DataComparator interface {
	CompareData(name DataName, data1, data2 Data) (bool, error)
}

type DefaultDataComparator struct {
}

func (c *DefaultDataComparator) CompareData(name DataName, data1, data2 Data) (bool, error) {
	if data1.DataType() != data2.DataType() {
		return false, nil
	}
	return fmt.Sprintf("%s", data1) == fmt.Sprintf("%s", data2), nil
}

type ValueComparator interface {
	CompareValue(key Key, value1, value2 Value) (ValueDiff, error)
}

type DefaultValueComparator struct {
	DataComparator DataComparator
	IgnoreKeys     []Key
}

func (c *DefaultValueComparator) CompareValue(key Key, value1, value2 Value) (ValueDiff, error) {
	diff := NewValueDiff()
	for _, ignoreKey := range c.IgnoreKeys {
		if key == ignoreKey {
			return diff, nil
		}
	}
	for dataName, data1 := range value1 {
		data2, ok := value2[dataName]
		if !ok {
			diff.Value1[dataName] = data1
			continue
		}
		equals, err := c.DataComparator.CompareData(dataName, data1, data2)
		if err != nil {
			return diff, err
		}
		if !equals {
			diff.Value1[dataName] = data1
			diff.Value2[dataName] = data2
		}
	}
	for dataName, data2 := range value2 {
		if _, ok := value1[dataName]; !ok {
			diff.Value2[dataName] = data2
		}
	}
	return diff, nil
}
