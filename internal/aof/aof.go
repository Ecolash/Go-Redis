package aof

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
