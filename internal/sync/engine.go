package sync

import (
	"context"
	"fmt"
	"os"

	"simpler2sync/internal/r2client"
	"simpler2sync/internal/store"
)

type SyncResult struct {
	TaskID  string
	Success int
	Failed  int
}

func RunSync(ctx context.Context, client *r2client.Client, st *store.Store, taskID, localRoot, bucket, remotePrefix, conflictStrategy string, cb ExecCallback) (*SyncResult, error) {
	if _, err := os.Stat(localRoot); os.IsNotExist(err) {
		return nil, fmt.Errorf("local path does not exist: %s", localRoot)
	}
	remotePrefix = r2client.NormalizePrefix(remotePrefix)

	localFiles, err := WalkLocal(localRoot)
	if err != nil {
		return nil, fmt.Errorf("walk local: %w", err)
	}

	remoteObjects, err := ListRemote(ctx, client, bucket, remotePrefix)
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}

	states, err := st.ListStatesForTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("load states: %w", err)
	}

	actions, err := Diff(localFiles, remoteObjects, stateMap(states), taskID, localRoot, remotePrefix, conflictStrategy)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}

	skippedFailed := 0
	resolvedActions := actions[:0]
	for i := range actions {
		if actions[i].LocalPath == "" {
			localPath, err := localPathForRemoteKey(localRoot, remotePrefix, actions[i].RemoteKey)
			if err != nil {
				skippedFailed++
				if cb != nil {
					cb(actions[i], i+1, len(actions), err)
				}
				continue
			}
			actions[i].LocalPath = localPath
		}
		resolvedActions = append(resolvedActions, actions[i])
	}
	actions = resolvedActions

	success, failed, err := Execute(ctx, client, st, bucket, localRoot, actions, taskID, conflictStrategy, cb)
	if err != nil {
		return nil, err
	}
	failed += skippedFailed

	return &SyncResult{
		TaskID:  taskID,
		Success: success,
		Failed:  failed,
	}, nil
}
