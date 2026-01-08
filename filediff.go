package wineregdiff

import (
	"fmt"
	"sort"
	"strings"
)

func GenerateRegFile(diff RegistryDiff, root RegistryRoot, changesFor ChangesFor) string {
	var sb strings.Builder
	sb.WriteString("Windows Registry Editor Version 5.00\r\n")

	addDiff := diff.Registry2Only
	deleteDiff := diff.Registry1Only
	if changesFor == ChangesFor2 {
		addDiff, deleteDiff = deleteDiff, addDiff
	}

	// Collect and sort keys for consistent output
	var addKeys []Key
	for key := range addDiff {
		addKeys = append(addKeys, key)
	}
	sort.Slice(addKeys, func(i, j int) bool { return addKeys[i] < addKeys[j] })

	var deleteKeys []Key
	for key := range deleteDiff {
		deleteKeys = append(deleteKeys, key)
	}
	sort.Slice(deleteKeys, func(i, j int) bool { return deleteKeys[i] < deleteKeys[j] })

	var changedKeys []Key
	for key := range diff.RegistryChanged {
		changedKeys = append(changedKeys, key)
	}
	sort.Slice(changedKeys, func(i, j int) bool { return changedKeys[i] < changedKeys[j] })

	// Add new keys/values
	for _, key := range addKeys {
		value := addDiff[key]
		sb.WriteString("\r\n")
		writeRegFileKey(&sb, root, key, value)
	}

	// Delete keys/values
	for _, key := range deleteKeys {
		value := deleteDiff[key]
		sb.WriteString("\r\n")
		writeRegFileDeleteKey(&sb, root, key, value)
	}

	// Changed values
	for _, key := range changedKeys {
		changed := diff.RegistryChanged[key]
		var newValue, oldValue Value
		if changesFor == ChangesFor1 {
			newValue = changed.Value2
			oldValue = changed.Value1
		} else {
			newValue = changed.Value1
			oldValue = changed.Value2
		}
		sb.WriteString("\r\n")
		writeRegFileChangedKey(&sb, root, key, newValue, oldValue)
	}

	return sb.String()
}

func writeRegFileKey(sb *strings.Builder, root RegistryRoot, key Key, value Value) {
	keyPath := fmt.Sprintf(`%s\%s`, root, key)
	sb.WriteString(fmt.Sprintf("[%s]\r\n", keyPath))
	writeRegFileValues(sb, value)
}

func writeRegFileDeleteKey(sb *strings.Builder, root RegistryRoot, key Key, value Value) {
	keyPath := fmt.Sprintf(`%s\%s`, root, key)
	if len(value) == 0 {
		// Delete entire key
		sb.WriteString(fmt.Sprintf("[-%s]\r\n", keyPath))
		return
	}
	// Delete specific values
	sb.WriteString(fmt.Sprintf("[%s]\r\n", keyPath))
	dataNames := sortedDataNames(value)
	for _, dataName := range dataNames {
		if dataName == UnnamedDataName {
			sb.WriteString("@=-\r\n")
		} else {
			sb.WriteString(fmt.Sprintf("\"%s\"=-\r\n", dataName))
		}
	}
}

func writeRegFileChangedKey(sb *strings.Builder, root RegistryRoot, key Key, newValue, oldValue Value) {
	keyPath := fmt.Sprintf(`%s\%s`, root, key)
	sb.WriteString(fmt.Sprintf("[%s]\r\n", keyPath))

	// Delete old values that are not in new
	oldDataNames := sortedDataNames(oldValue)
	for _, dataName := range oldDataNames {
		if _, ok := newValue[dataName]; !ok {
			if dataName == UnnamedDataName {
				sb.WriteString("@=-\r\n")
			} else {
				sb.WriteString(fmt.Sprintf("\"%s\"=-\r\n", dataName))
			}
		}
	}

	// Add/update new values
	writeRegFileValues(sb, newValue)
}

func writeRegFileValues(sb *strings.Builder, value Value) {
	dataNames := sortedDataNames(value)
	for _, dataName := range dataNames {
		data := value[dataName]
		regFileValue := data.RegFileString()
		if dataName == UnnamedDataName {
			sb.WriteString(fmt.Sprintf("@=%s\r\n", regFileValue))
		} else {
			sb.WriteString(fmt.Sprintf("\"%s\"=%s\r\n", dataName, regFileValue))
		}
	}
}

func sortedDataNames(value Value) []DataName {
	var dataNames []DataName
	for dataName := range value {
		dataNames = append(dataNames, dataName)
	}
	sort.Slice(dataNames, func(i, j int) bool { return dataNames[i] < dataNames[j] })
	return dataNames
}
