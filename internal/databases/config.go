package databases

import (
	"os"
	"path/filepath"
	"strings"
)

// DataDir returns the directory used for cached IEEE/IANA files and related data.
// Override with CERBERUS_DATA_DIR (absolute or relative path).
func DataDir() string {
	if d := strings.TrimSpace(os.Getenv("CERBERUS_DATA_DIR")); d != "" {
		return filepath.Clean(d)
	}
	return filepath.Clean("./data")
}

// OnlineDB reports whether automatic downloads of IEEE OUI and IANA services
// are allowed when local cache is missing or stale. Set CERBERUS_DB_ONLINE=1,
// true, or yes.
func OnlineDB() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CERBERUS_DB_ONLINE")))
	return v == "1" || v == "true" || v == "yes"
}
