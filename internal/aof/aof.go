package aof

import (
	"fmt"
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

// Setup creates the append-only directory (dir/appendDirName) containing an
// empty first incremental AOF file (appendFilename.1.incr.aof) and a manifest
// file (appendFilename.manifest) describing it. If the manifest already exists
// (i.e. this is a restart with persisted data), Setup does nothing beyond
// ensuring the directory exists.
func Setup(dir, appendDirName, appendFilename string) error {
	aofDir := filepath.Join(dir, appendDirName)
	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		return err
	}
	manifestPath := filepath.Join(aofDir, appendFilename+".manifest")
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	}
	incrName := appendFilename + ".1.incr.aof"
	f, err := os.OpenFile(filepath.Join(aofDir, incrName), os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	manifest := fmt.Sprintf("file %s seq 1 type i\n", incrName)
	return os.WriteFile(manifestPath, []byte(manifest), 0o644)
}
