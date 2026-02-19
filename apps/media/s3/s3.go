package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/getevo/dsn"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Timeout is the default deadline for every S3 API call.
// Prevents goroutine leaks if the storage backend is unresponsive.
const s3Timeout = 30 * time.Second

// FileSystem implements filesystem.Interface using minio-go, which is fully
// compatible with GCS HMAC keys, AWS S3, MinIO, Cloudflare R2, and other
// S3-compatible services. Unlike the AWS SDK v2, minio-go does not inject
// x-amz-checksum-* or x-amz-sdk-* headers that cause GCS to return
// SignatureDoesNotMatch errors.
//
// DSN format:
//
//	s3://ACCESS_KEY:SECRET_KEY@ENDPOINT/BUCKET?Region=auto&IgnoreSSL=false
//
// Notable DSN params:
//
//	Region    – signing region (default: us-east-1; use "auto" for GCS/R2)
//	IgnoreSSL – skip TLS verification (default: false)
type FileSystem struct {
	DSN       string `dsn:"s3://$AccessKey:$SecretKey@$Endpoint/$Bucket"`
	Scheme    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	BasePath  string `default:""`
	IgnoreSSL bool   `default:"false"`
	PathStyle bool   `default:"false"`
	Params    map[string]string

	client *minio.Client
}

// New creates and initialises a FileSystem from a DSN string.
func New(configString string) (*FileSystem, error) {
	f := &FileSystem{}
	if err := f.Setup(configString); err != nil {
		return nil, err
	}
	return f, nil
}

func (l *FileSystem) Setup(confString string) error {
	if err := dsn.ParseDSN(confString, l); err != nil {
		return fmt.Errorf("failed to parse S3 DSN: %w", err)
	}

	region := l.Region
	if region == "" {
		region = "us-east-1"
	}

	useSSL := !l.IgnoreSSL

	// Default to path-style for non-AWS endpoints (iDrive, MinIO, Cloudflare R2, etc.).
	// AWS S3 uses virtual-hosted style; everything else typically requires path-style.
	isAWS := strings.HasSuffix(l.Endpoint, "amazonaws.com")
	lookup := minio.BucketLookupPath
	if isAWS && !l.PathStyle {
		lookup = minio.BucketLookupAuto
	}

	var err error
	l.client, err = minio.New(l.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(l.AccessKey, l.SecretKey, ""),
		Secure:       useSSL,
		Region:       region,
		BucketLookup: lookup,
	})
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Verify bucket is accessible.
	ctx, cancel := l.newCtx()
	defer cancel()
	exists, err := l.client.BucketExists(ctx, l.Bucket)
	if err != nil {
		return fmt.Errorf("S3 bucket check failed for %q: %w", l.Bucket, err)
	}
	if !exists {
		return fmt.Errorf("S3 bucket %q does not exist", l.Bucket)
	}

	return nil
}

// newCtx returns a context with s3Timeout deadline for a single S3 API call.
func (l *FileSystem) newCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s3Timeout)
}

// joinKey builds an S3 object key from the base path and a relative path,
// always using forward slashes.
func (l *FileSystem) joinKey(p string) string {
	if l.BasePath == "" {
		return strings.TrimPrefix(path.Clean("/"+p), "/")
	}
	return strings.TrimPrefix(path.Join(l.BasePath, p), "/")
}

// ── filesystem.Interface implementation ──────────────────────────────────────

func (l *FileSystem) Touch(p string) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.PutObject(ctx, l.Bucket, l.joinKey(p),
		bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	return err
}

func (l *FileSystem) Delete(p string) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	return l.client.RemoveObject(ctx, l.Bucket, l.joinKey(p), minio.RemoveObjectOptions{})
}

func (l *FileSystem) List(p string) ([]string, error) {
	prefix := l.joinKey(p)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := l.newCtx()
	defer cancel()
	var result []string
	for obj := range l.client.ListObjects(ctx, l.Bucket, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		result = append(result, strings.TrimPrefix(obj.Key, prefix))
	}
	return result, nil
}

func (l *FileSystem) Walk(p string, fn func(path string, info fs.FileInfo, err error) error) error {
	prefix := l.joinKey(p)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := l.newCtx()
	defer cancel()
	for obj := range l.client.ListObjects(ctx, l.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return fn("", nil, obj.Err)
		}
		fi := &fileInfo{key: obj.Key, size: obj.Size, mod: obj.LastModified}
		relPath := strings.TrimPrefix(obj.Key, prefix)
		if err := fn(relPath, fi, nil); err != nil {
			return err
		}
	}
	return nil
}

func (l *FileSystem) Read(p string) ([]byte, error) {
	ctx, cancel := l.newCtx()
	defer cancel()
	obj, err := l.client.GetObject(ctx, l.Bucket, l.joinKey(p), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (l *FileSystem) IsDir(p string) (bool, error) {
	prefix := l.joinKey(p)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := l.newCtx()
	defer cancel()
	for obj := range l.client.ListObjects(ctx, l.Bucket, minio.ListObjectsOptions{Prefix: prefix, MaxKeys: 1}) {
		if obj.Err != nil {
			return false, obj.Err
		}
		return true, nil
	}
	return false, nil
}

func (l *FileSystem) IsFile(p string) (bool, error) {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.StatObject(ctx, l.Bucket, l.joinKey(p), minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *FileSystem) Mkdir(p string) error {
	key := l.joinKey(p)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.PutObject(ctx, l.Bucket, key,
		bytes.NewReader([]byte{}), 0, minio.PutObjectOptions{})
	return err
}

func (l *FileSystem) Write(p string, data []byte) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.PutObject(ctx, l.Bucket, l.joinKey(p),
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return err
}

func (l *FileSystem) WriteBuffer(p string, reader io.Reader) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.PutObject(ctx, l.Bucket, l.joinKey(p),
		reader, -1, minio.PutObjectOptions{})
	return err
}

func (l *FileSystem) Exists(p string) (bool, error) {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.StatObject(ctx, l.Bucket, l.joinKey(p), minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *FileSystem) Stat(p string) (fs.FileInfo, error) {
	key := l.joinKey(p)
	ctx, cancel := l.newCtx()
	defer cancel()
	info, err := l.client.StatObject(ctx, l.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &fileInfo{key: key, size: info.Size, mod: info.LastModified}, nil
}

func (l *FileSystem) Copy(src, dst string) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	srcKey := l.joinKey(src)
	dstKey := l.joinKey(dst)
	_, err := l.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: l.Bucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: l.Bucket, Object: srcKey},
	)
	return err
}

func (l *FileSystem) Move(src, dst string) error {
	if err := l.Copy(src, dst); err != nil {
		return err
	}
	return l.Delete(src)
}

func (l *FileSystem) DiskToStorage(src, dst string) error {
	ctx, cancel := l.newCtx()
	defer cancel()
	_, err := l.client.FPutObject(ctx, l.Bucket, l.joinKey(dst), src, minio.PutObjectOptions{})
	return err
}

func (l *FileSystem) StorageToDisk(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	ctx, cancel := l.newCtx()
	defer cancel()
	return l.client.FGetObject(ctx, l.Bucket, l.joinKey(src), dst, minio.GetObjectOptions{})
}

// ── fs.FileInfo implementation ────────────────────────────────────────────────

type fileInfo struct {
	key  string
	size int64
	mod  time.Time
}

func (fi *fileInfo) Name() string       { return path.Base(fi.key) }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() fs.FileMode  { return 0444 }
func (fi *fileInfo) ModTime() time.Time { return fi.mod }
func (fi *fileInfo) IsDir() bool        { return strings.HasSuffix(fi.key, "/") }
func (fi *fileInfo) Sys() interface{}   { return nil }
