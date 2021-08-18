package wineregdiff

type Registry map[Key]Value
type Key string
type Value map[DataName]Data
type DataName string
type RegistryRoot string

const (
	RegistryRootLocalMachine  RegistryRoot = "HKEY_LOCAL_MACHINE"
	RegistryRootCurrentUser   RegistryRoot = "HKEY_CURRENT_USER"
	RegistryRootClassesRoot   RegistryRoot = "HKEY_CLASSES_ROOT"
	RegistryRootUsers         RegistryRoot = "HKEY_USERS"
	RegistryRootCurrentConfig RegistryRoot = "HKEY_CURRENT_CONFIG"

	UnnamedDataName DataName = "@"
)
