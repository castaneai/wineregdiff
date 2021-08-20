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

type RegCommand struct {
	Args []string
}

func GenerateRegCommands(diff RegistryDiff, root RegistryRoot, changesFor ChangesFor, force bool) []RegCommand {
	var cmds []RegCommand
	addDiff := diff.Registry2Only
	deleteDiff := diff.Registry1Only
	if changesFor == ChangesFor2 {
		addDiff, deleteDiff = deleteDiff, addDiff
	}
	for key, value := range addDiff {
		cmds = append(cmds, addCommand(root, key, value, force))
	}
	for key, value := range deleteDiff {
		cmds = append(cmds, deleteCommand(root, key, value, force))
	}
	for key, changed := range diff.RegistryChanged {
		switch changesFor {
		case ChangesFor1:
			cmds = append(cmds, addCommand(root, key, changed.Value2, force))
		case ChangesFor2:
			cmds = append(cmds, addCommand(root, key, changed.Value1, force))
		}
	}
	return cmds
}

func addCommand(root RegistryRoot, key Key, value Value, force bool) RegCommand {
	cmd := RegCommand{
		Args: []string{"REG", "ADD"},
	}
	keyName := escapeString(fmt.Sprintf(`%s\%s`, root, key))
	cmd.Args = append(cmd.Args, fmt.Sprintf(`"%s"`, keyName))
	if len(value) == 0 {
		return cmd
	}
	for dataName, data := range value {
		v := "/v"
		if dataName == UnnamedDataName {
			v = "/ve"
		}
		cmd.Args = append(cmd.Args, []string{
			v, fmt.Sprintf(`"%s"`, dataName),
			fmt.Sprintf("/t"), data.DataType().String(),
			"/d", fmt.Sprintf(`"%s"`, data.CommandString()),
		}...)
		if force {
			cmd.Args = append(cmd.Args, "/f")
		}
	}
	return cmd
}

func deleteCommand(root RegistryRoot, key Key, value Value, force bool) RegCommand {
	cmd := RegCommand{
		Args: []string{"REG", "DELETE"},
	}
	keyName := escapeString(fmt.Sprintf(`%s\%s`, root, key))
	cmd.Args = append(cmd.Args, fmt.Sprintf(`"%s"`, keyName))
	if len(value) == 0 {
		return cmd
	}
	for dataName := range value {
		v := "/v"
		if dataName == UnnamedDataName {
			v = "/ve"
		}
		cmd.Args = append(cmd.Args, []string{
			v, fmt.Sprintf(`"%s"`, dataName),
		}...)
		if force {
			cmd.Args = append(cmd.Args, "/f")
		}
	}
	return cmd
}
