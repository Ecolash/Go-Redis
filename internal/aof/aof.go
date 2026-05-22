package aof

import (
	"os"
	"path/filepath"
)

// Default values for AOF (Append Only File) persistence configuration options.
// These are surfaced via CONFIG GET; no persistence logic is implemented yet.
const (
	DefaultAppendOnly     = "no"
	DefaultAppendDirName  = "appendonlydir"
	DefaultAppendFilename = "appendonly.aof"
	DefaultAppendFsync    = "everysec"
)

// Defaults returns the AOF-related config options and their default values.
func Defaults() map[string]string {
	return map[string]string{
		"appendonly":     DefaultAppendOnly,
		"appenddirname":  DefaultAppendDirName,
		"appendfilename": DefaultAppendFilename,
		"appendfsync":    DefaultAppendFsync,
	}
}

// Setup creates the append-only directory (dir/appendDirName) and an empty
// first incremental AOF file (appendFilename.1.incr.aof) within it. Existing
// directories and files are left intact.
func Setup(dir, appendDirName, appendFilename string) error {
	aofDir := filepath.Join(dir, appendDirName)
	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		return err
	}
	incrPath := filepath.Join(aofDir, appendFilename+".1.incr.aof")
	f, err := os.OpenFile(incrPath, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
