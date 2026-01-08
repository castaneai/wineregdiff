package wineregdiff

// FileMetadata holds file-level metadata from Wine registry files
type FileMetadata struct {
	// Comment is the file-level comment (e.g., ";; All keys relative to REGISTRY\\Machine")
	Comment string
	// Arch is the architecture (e.g., "win64")
	Arch string
}

// KeyMetadata holds key-level metadata
type KeyMetadata struct {
	// Timestamp is the Unix timestamp for the key
	Timestamp int64
	// Time is the hex time value (e.g., "1dc6e34610bc106")
	Time string
}

// RegistryFile represents a complete Wine registry file with metadata
type RegistryFile struct {
	// FileMetadata holds file-level metadata
	FileMetadata FileMetadata
	// Registry holds the actual registry data
	Registry Registry
	// KeyMetadata holds per-key metadata (key -> metadata)
	KeyMetadata map[Key]KeyMetadata
}

// NewRegistryFile creates a new RegistryFile
func NewRegistryFile() *RegistryFile {
	return &RegistryFile{
		Registry:    make(Registry),
		KeyMetadata: make(map[Key]KeyMetadata),
	}
}

// Merge merges other RegistryFile into rf.
// Registry data is merged (same key/value names are overwritten).
// KeyMetadata from other overwrites existing metadata for the same keys.
func (rf *RegistryFile) Merge(other *RegistryFile) {
	rf.Registry.Merge(other.Registry)
	for key, meta := range other.KeyMetadata {
		rf.KeyMetadata[key] = meta
	}
}

// Clone creates a deep copy of the RegistryFile
func (rf *RegistryFile) Clone() *RegistryFile {
	clone := NewRegistryFile()
	clone.FileMetadata = rf.FileMetadata
	clone.Registry = rf.Registry.Clone()
	for key, meta := range rf.KeyMetadata {
		clone.KeyMetadata[key] = meta
	}
	return clone
}
