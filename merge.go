package wineregdiff

// Merge merges the contents of other into r.
// If the same key/value name exists, it will be overwritten with the value from other.
func (r Registry) Merge(other Registry) {
	for key, otherValue := range other {
		if existingValue, ok := r[key]; ok {
			// Key exists, merge values
			for dataName, data := range otherValue {
				existingValue[dataName] = data
			}
		} else {
			// Key doesn't exist, add new
			r[key] = make(Value)
			for dataName, data := range otherValue {
				r[key][dataName] = data
			}
		}
	}
}

// Clone creates a deep copy of the Registry
func (r Registry) Clone() Registry {
	clone := make(Registry)
	for key, value := range r {
		clone[key] = make(Value)
		for dataName, data := range value {
			clone[key][dataName] = data
		}
	}
	return clone
}
