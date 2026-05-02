package sync

import (
	"fmt"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type ActionType string

const (
	ActionUpload   ActionType = "upload"
	ActionDownload ActionType = "download"
)

type Action struct {
	Type      ActionType
	LocalPath string
	RemoteKey string
}

func Diff(localFiles []FileInfo, remoteObjects map[string]r2client.ObjectInfo, states map[string]*store.SyncState, taskID, localRoot, remotePrefix string, conflictStrategy string) ([]Action, error) {
	var actions []Action
	localMap := make(map[string]FileInfo)
	for _, f := range localFiles {
		key, err := r2client.RemoteKey(localRoot, f.Path, remotePrefix)
		if err != nil {
			return nil, fmt.Errorf("remote key: %w", err)
		}
		localMap[key] = f
	}
	remoteKeys := make(map[string]bool)
	for key := range remoteObjects {
		remoteKeys[key] = true
	}

	for key, local := range localMap {
		remote, hasRemote := remoteObjects[key]
		st := states[key]
		if !hasRemote {
			actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
		} else if hasRemote && st != nil {
			localChanged := local.Hash != st.LocalHash
			remoteChanged := remote.ETag != st.RemoteETag
			if localChanged && !remoteChanged {
				actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
			} else if !localChanged && remoteChanged {
				actions = append(actions, Action{Type: ActionDownload, LocalPath: local.Path, RemoteKey: key})
			} else if localChanged && remoteChanged {
				switch conflictStrategy {
				case "newer":
					if local.ModTime > st.RemoteMtime {
						actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
					} else {
						actions = append(actions, Action{Type: ActionDownload, LocalPath: local.Path, RemoteKey: key})
					}
				case "mirror":
					actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
				}
			}
		} else {
			actions = append(actions, Action{Type: ActionUpload, LocalPath: local.Path, RemoteKey: key})
		}
		delete(remoteKeys, key)
	}

	for key := range remoteKeys {
		if _, hasLocal := localMap[key]; !hasLocal {
			actions = append(actions, Action{Type: ActionDownload, LocalPath: "", RemoteKey: key})
		}
	}

	return actions, nil
}

func stateMap(states []store.SyncState) map[string]*store.SyncState {
	m := make(map[string]*store.SyncState)
	for i := range states {
		m[states[i].RemoteKey] = &states[i]
	}
	return m
}
