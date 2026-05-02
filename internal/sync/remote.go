package sync

import (
	"context"
	"fmt"

	"simpler2sync/internal/r2client"
)

func ListRemote(ctx context.Context, client *r2client.Client, bucket, prefix string) (map[string]r2client.ObjectInfo, error) {
	objects, err := client.ListObjects(ctx, bucket, prefix)
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}
	result := make(map[string]r2client.ObjectInfo, len(objects))
	for _, obj := range objects {
		result[obj.Key] = obj
	}
	return result, nil
}
