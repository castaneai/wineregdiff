package wineregdiff

type RegistryDiff struct {
	Registry1Only   Registry
	Registry2Only   Registry
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
	Value1 Value
	Value2 Value
}

func (d ValueDiff) HasDiff() bool {
	return len(d.Value1) > 0 || len(d.Value2) > 0
}

func NewValueDiff() ValueDiff {
	return ValueDiff{
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
