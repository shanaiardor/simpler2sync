package sync

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"simpler2sync/internal/r2client"
)

func localPathForRemoteKey(localRoot, remotePrefix, remoteKey string) (string, error) {
	key := strings.ReplaceAll(filepath.ToSlash(remoteKey), `\`, "/")
	prefix := r2client.NormalizePrefix(remotePrefix)
	if prefix != "" {
		if !strings.HasPrefix(key, prefix) {
			return "", fmt.Errorf("remote key %q is outside prefix %q", remoteKey, prefix)
		}
		key = strings.TrimPrefix(key, prefix)
	}

	rel := path.Clean(strings.TrimLeft(key, "/"))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("unsafe remote key: %q", remoteKey)
	}

	localRoot = filepath.Clean(localRoot)
	localPath := filepath.Clean(filepath.Join(localRoot, filepath.FromSlash(rel)))
	back, err := filepath.Rel(localRoot, localPath)
	if err != nil {
		return "", fmt.Errorf("rel local path: %w", err)
	}
	if back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) || filepath.IsAbs(back) {
		return "", fmt.Errorf("remote key escapes local root: %q", remoteKey)
	}

	return localPath, nil
}
