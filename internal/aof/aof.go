package aof

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultAppendOnly     = "no"
	DefaultAppendDirName  = "appendonlydir"
	DefaultAppendFilename = "appendonly.aof"
	DefaultAppendFsync    = "everysec"
)

func Defaults() map[string]string {
	return map[string]string{
		"appendonly":     DefaultAppendOnly,
		"appenddirname":  DefaultAppendDirName,
		"appendfilename": DefaultAppendFilename,
		"appendfsync":    DefaultAppendFsync,
	}
}

func Setup(dir, appendDirName, appendFilename string) error {
	aofDir := filepath.Join(dir, appendDirName)
	manifestPath := filepath.Join(aofDir, appendFilename+".manifest")
	incrName := appendFilename + ".1.incr.aof"

	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	}

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
