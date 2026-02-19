package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/getevo/dsn"
)

// FileSystem is a local S3/GCS-compatible implementation that fixes the upstream
// filesystem/s3 package issues:
//
//  1. AWS SDK v2 ≥ v1.32 sends x-amz-checksum-* headers by default
//     (RequestChecksumCalculationWhenSupported). GCS, MinIO, and many other
//     S3-compatible services reject these headers. We force WhenRequired.
//
//  2. The upstream package hard-codes UsePathStyle=true. Some providers require
//     virtual-hosted style (PathStyle=false). Configurable via DSN param.
//
//  3. Endpoint scheme: the DSN strips the s3:// scheme, so we re-add https://
//     (or http:// when IgnoreSSL=true) when no scheme is present.
//
//  4. filepath.Join uses OS path separator (\ on Windows). S3 keys must use
//     forward slashes, so we use path.Join for key construction.
type FileSystem struct {
	DSN       string `dsn:"s3://$AccessKey:$SecretKey@$Endpoint/$Bucket"`
	Scheme    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	BasePath  string `default:""`
	Debug     bool   `default:"false"`
	IgnoreSSL bool   `default:"false"`
	Params    map[string]string

	pathStyle     bool
	configuration aws.Config
	client        *s3.Client
}

// New creates and initialises a FileSystem from a DSN string.
//
// DSN format:
//
//	s3://ACCESS_KEY:SECRET_KEY@ENDPOINT/BUCKET?Region=us-east-1&PathStyle=true&Timeout=60s&IgnoreSSL=false
//
// Notable DSN params:
//
//	Region    – AWS/GCS region (default: us-east-1; use "auto" for Cloudflare R2)
//	PathStyle – true (default) uses path-style addressing; false uses virtual-hosted
//	Timeout   – HTTP timeout duration (default: 60s)
//	IgnoreSSL – skip TLS certificate verification (default: false)
func New(configString string) (*FileSystem, error) {
	fs := &FileSystem{}
	if err := fs.Setup(configString); err != nil {
		return nil, err
	}
	return fs, nil
}

func (l *FileSystem) Setup(confString string) error {
	if err := dsn.ParseDSN(confString, l); err != nil {
		return fmt.Errorf("failed to parse S3 DSN: %w", err)
	}

	// PathStyle defaults to true for backward compatibility.
	l.pathStyle = true
	if v, ok := l.Params["PathStyle"]; ok {
		l.pathStyle = strings.ToLower(v) != "false"
	}

	// Region: DSN struct-field takes precedence; Params override if not set by struct.
	if l.Region == "" {
		l.Region = "us-east-1"
	}
	if v, ok := l.Params["Region"]; ok && v != "" {
		l.Region = v
	}

	// Timeout
	timeout := 60 * time.Second
	if v, ok := l.Params["Timeout"]; ok {
		var err error
		timeout, err = time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("unable to parse S3 Timeout param: %v", err)
		}
	}

	// Normalize endpoint: ensure it has a scheme so the SDK can build URLs.
	endpoint := l.Endpoint
	if endpoint != "" &&
		!strings.HasPrefix(endpoint, "http://") &&
		!strings.HasPrefix(endpoint, "https://") {
		if l.IgnoreSSL {
			endpoint = "http://" + endpoint
		} else {
			endpoint = "https://" + endpoint
		}
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: l.IgnoreSSL, //nolint:gosec // user opt-in
			},
		},
		Timeout: timeout,
	}

	var err error
	l.configuration, err = config.LoadDefaultConfig(context.TODO(),
		config.WithHTTPClient(httpClient),
		config.WithRegion(l.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(l.AccessKey, l.SecretKey, ""),
		),
		// Critical for GCS / MinIO / R2 / non-AWS providers:
		// AWS SDK v2 ≥ v1.32 sends x-amz-checksum-* headers proactively by default.
		// GCS and many other S3-compatible services do not understand these headers
		// and return errors. Setting both options to WhenRequired means checksums are
		// only added when the server explicitly requests them.
		config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
		config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
	)
	if err != nil {
		return fmt.Errorf("failed to load S3 config: %w", err)
	}

	l.client = s3.NewFromConfig(l.configuration, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = l.pathStyle
	})

	// Verify connectivity and bucket access.
	_, err = l.client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(l.Bucket),
	})
	if err != nil {
		return fmt.Errorf("S3 bucket %q not accessible at %s: %w", l.Bucket, endpoint, err)
	}

	return nil
}

// joinKey builds an S3 object key from the filesystem base path and a relative
// path, always using forward slashes regardless of the host OS.
func (l *FileSystem) joinKey(p string) string {
	if l.BasePath == "" {
		return strings.TrimPrefix(path.Clean("/"+p), "/")
	}
	return strings.TrimPrefix(path.Join(l.BasePath, p), "/")
}

// ── filesystem.Interface implementation ──────────────────────────────────────

func (l *FileSystem) Touch(p string) error {
	_, err := l.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
		Body:   bytes.NewReader([]byte{}),
	})
	return err
}

func (l *FileSystem) Delete(p string) error {
	_, err := l.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
	})
	return err
}

func (l *FileSystem) List(p string) ([]string, error) {
	prefix := l.joinKey(p)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var result []string
	paginator := s3.NewListObjectsV2Paginator(l.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(l.Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			result = append(result, strings.TrimPrefix(*obj.Key, prefix))
		}
	}
	return result, nil
}

func (l *FileSystem) Walk(p string, fn func(path string, info fs.FileInfo, err error) error) error {
	prefix := l.joinKey(p)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	paginator := s3.NewListObjectsV2Paginator(l.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(l.Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fn("", nil, err)
		}
		for _, obj := range page.Contents {
			fi := &fileInfo{
				key:  *obj.Key,
				size: *obj.Size,
				mod:  *obj.LastModified,
			}
			relPath := strings.TrimPrefix(*obj.Key, prefix)
			if err := fn(relPath, fi, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *FileSystem) Read(p string) ([]byte, error) {
	out, err := l.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (l *FileSystem) IsDir(p string) (bool, error) {
	prefix := l.joinKey(p)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	out, err := l.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(l.Bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, err
	}
	return len(out.Contents) > 0, nil
}

func (l *FileSystem) IsFile(p string) (bool, error) {
	_, err := l.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
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
	_, err := l.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte{}),
	})
	return err
}

func (l *FileSystem) Write(p string, data []byte) error {
	_, err := l.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (l *FileSystem) WriteBuffer(p string, reader io.Reader) error {
	_, err := l.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
		Body:   reader,
	})
	return err
}

func (l *FileSystem) Exists(p string) (bool, error) {
	_, err := l.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(p)),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *FileSystem) Stat(p string) (fs.FileInfo, error) {
	key := l.joinKey(p)
	head, err := l.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return &fileInfo{
		key:  key,
		size: *head.ContentLength,
		mod:  *head.LastModified,
	}, nil
}

func (l *FileSystem) Copy(src, dst string) error {
	srcKey := l.joinKey(src)
	dstKey := l.joinKey(dst)
	copySource := l.Bucket + "/" + srcKey
	_, err := l.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(l.Bucket),
		CopySource: aws.String(url.PathEscape(copySource)),
		Key:        aws.String(dstKey),
	})
	return err
}

func (l *FileSystem) Move(src, dst string) error {
	if err := l.Copy(src, dst); err != nil {
		return err
	}
	return l.Delete(src)
}

func (l *FileSystem) DiskToStorage(src, dst string) error {
	const threshold = 50 * 1024 * 1024
	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("unable to stat local file: %w", err)
	}
	if fi.Size() < threshold {
		return l.PutFile(src, dst)
	}
	return l.MultipartUploadFile(src, dst)
}

func (l *FileSystem) StorageToDisk(src, dst string) error {
	resp, err := l.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(src)),
	})
	if err != nil {
		return fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write downloaded file: %w", err)
	}
	return nil
}

func (l *FileSystem) PutFile(localPath, s3Path string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = l.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(l.joinKey(s3Path)),
		Body:   f,
	})
	return err
}

func (l *FileSystem) MultipartUploadFile(localPath, s3Path string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	key := l.joinKey(s3Path)

	createResp, err := l.client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
		Bucket: aws.String(l.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	const partSize int64 = 5 * 1024 * 1024 // 5 MB minimum
	numParts := int((stat.Size() + partSize - 1) / partSize)
	completedParts := make([]types.CompletedPart, 0, numParts)

	for i := 0; i < numParts; i++ {
		partNum := int32(i + 1)
		start := int64(i) * partSize
		end := start + partSize
		if end > stat.Size() {
			end = stat.Size()
		}

		buf := make([]byte, end-start)
		if _, err := f.ReadAt(buf, start); err != nil {
			l.abortMultipart(key, createResp.UploadId)
			return fmt.Errorf("failed reading part %d: %w", partNum, err)
		}

		upResp, err := l.client.UploadPart(context.TODO(), &s3.UploadPartInput{
			Bucket:     aws.String(l.Bucket),
			Key:        aws.String(key),
			PartNumber: aws.Int32(partNum),
			UploadId:   createResp.UploadId,
			Body:       bytes.NewReader(buf),
		})
		if err != nil {
			l.abortMultipart(key, createResp.UploadId)
			return fmt.Errorf("failed uploading part %d: %w", partNum, err)
		}

		completedParts = append(completedParts, types.CompletedPart{
			ETag:       upResp.ETag,
			PartNumber: aws.Int32(partNum),
		})
	}

	_, err = l.client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(l.Bucket),
		Key:      aws.String(key),
		UploadId: createResp.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	return err
}

func (l *FileSystem) abortMultipart(key string, uploadID *string) {
	_, _ = l.client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(l.Bucket),
		Key:      aws.String(key),
		UploadId: uploadID,
	})
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
