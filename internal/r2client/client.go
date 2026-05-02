package r2client

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client *s3.Client
}

type ObjectInfo struct {
	Key          string
	ETag         string
	Size         int64
	LastModified int64
}

func New(endpoint, accessKey, secretKey, region string) (*Client, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &Client{client: s3.NewFromConfig(cfg)}, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}
		for _, obj := range page.Contents {
			var lastMod int64
			if obj.LastModified != nil {
				lastMod = obj.LastModified.Unix()
			}
			objects = append(objects, ObjectInfo{
				Key:          aws.ToString(obj.Key),
				ETag:         strings.Trim(aws.ToString(obj.ETag), `"`),
				Size:         aws.ToInt64(obj.Size),
				LastModified: lastMod,
			})
		}
	}
	return objects, nil
}

func (c *Client) UploadFile(ctx context.Context, bucket, key string, reader io.Reader, size int64) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
	}
	result, err := c.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", key, err)
	}
	return strings.Trim(aws.ToString(result.ETag), `"`), nil
}

func (c *Client) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("download %s: %w", key, err)
	}
	return result.Body, aws.ToInt64(result.ContentLength), nil
}

func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

func RemoteKey(localRoot, filePath, remotePrefix string) (string, error) {
	rel, err := filepath.Rel(localRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	rel = filepath.ToSlash(rel)
	prefix := strings.TrimSuffix(NormalizePrefix(remotePrefix), "/")
	if prefix == "" {
		return rel, nil
	}
	return path.Join(prefix, rel), nil
}

func NormalizePrefix(prefix string) string {
	prefix = strings.Trim(toObjectKeyPath(prefix), "/")
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func toObjectKeyPath(value string) string {
	return strings.ReplaceAll(filepath.ToSlash(value), `\`, "/")
}
