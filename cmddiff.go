package wineregdiff

import "fmt"

type ChangesFor int

const (
	ChangesFor1 = 1
	ChangesFor2 = 2
)

func (c ChangesFor) String() string {
	switch c {
	case ChangesFor1:
		return "ChangesFor1"
	case ChangesFor2:
		return "ChangesFor2"
	default:
		return fmt.Sprintf("<unknown ChangesFor(%d)>", int(c))
	}
}

func GenerateRegCommands(diff RegistryDiff, root RegistryRoot, changesFor ChangesFor, force bool) []string {
	var cmds []string
	addDiff := diff.Registry2Only
	deleteDiff := diff.Registry1Only
	if changesFor == ChangesFor2 {
		addDiff, deleteDiff = deleteDiff, addDiff
	}
	for key, value := range addDiff {
		cmds = append(cmds, addCommand(root, key, value, force)...)
	}
	for key, value := range deleteDiff {
		cmds = append(cmds, deleteCommand(root, key, value, force)...)
	}
	for key, changed := range diff.RegistryChanged {
		switch changesFor {
		case ChangesFor1:
			cmds = append(cmds, addCommand(root, key, changed.Value2, force)...)
		case ChangesFor2:
			cmds = append(cmds, addCommand(root, key, changed.Value1, force)...)
		}
	}
	return cmds
}

func addCommand(root RegistryRoot, key Key, value Value, force bool) []string {
	var cmds []string
	keyName := escapeString(fmt.Sprintf(`%s\%s`, root, key))
	if len(value) == 0 {
		return []string{fmt.Sprintf(`REG ADD "%s"`, keyName)}
	}
	for dataName, data := range value {
		v := fmt.Sprintf(`/v "%s"`, dataName)
		if dataName == UnnamedDataName {
			v = `/ve`
		}
		cmd := fmt.Sprintf(`REG ADD "%s" %s /t %s /d "%s"`,
			keyName, v, data.DataType(), data.CommandString())
		if force {
			cmd += " /f"
		}
		cmds = append(cmds, cmd)
	}
	return cmds
}

func deleteCommand(root RegistryRoot, key Key, value Value, force bool) []string {
	var cmds []string
	keyName := escapeString(fmt.Sprintf(`%s\%s`, root, key))
	if len(value) == 0 {
		return []string{fmt.Sprintf(`REG DELETE "%s"`, keyName)}
	}
	for dataName := range value {
		v := fmt.Sprintf(`/v "%s"`, dataName)
		if dataName == UnnamedDataName {
			v = `/ve`
		}
		cmd := fmt.Sprintf(`REG DELETE "%s" %s`, keyName, v)
		if force {
			cmd += " /f"
		}
		cmds = append(cmds, cmd)
	}
	return cmds
}
