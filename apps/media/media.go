package media

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db/types"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/filesystem"
	"github.com/getevo/filesystem/http"
	"github.com/getevo/filesystem/localfs"
	localS3 "mediax/apps/media/s3"
	"github.com/getevo/restify"
	"github.com/gofiber/fiber/v2"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const STAGING = "__STAGING__"

type Type struct {
	Extension string
	Mime      string
	Encoders  map[string]*Encoder
}
type Options struct {
	Width           int
	Height          int
	KeepAspectRatio bool
	Quality         int
	CropDirection   string
	OutputFormat    string
	Profile         string
	Download        bool
	Encoder         *Encoder
	// Video-specific options
	Preview   string // "true", "480p", "720p", "1080p", "4k","wxy"
	Thumbnail string // "480p", "720p", "1080p", "4k"
	SS        int    // timestamp in seconds for thumbnail
	// Audio-specific options
	Detail bool // return JSON metadata when true
}

func (o Options) ToString() string {
	return fmt.Sprintf("%dx%da%tq%dd%sp%s", o.Width, o.Height, o.KeepAspectRatio, o.Quality, o.CropDirection, o.Profile)
}

// queryFirst returns the first non-empty value among the given query param names.
func queryFirst(request *evo.Request, names ...string) string {
	for _, name := range names {
		if v := request.Query(name).String(); v != "" {
			return v
		}
	}
	return ""
}

func (t *Type) ParseOptions(request *evo.Request) (*Options, error) {
	options := &Options{}

	// Accept both long form (width/height/format) and short aliases (w/h/f).
	if v := queryFirst(request, "width", "w"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid width value: %q", v)
		}
		options.Width = n
	}
	if v := queryFirst(request, "height", "h"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid height value: %q", v)
		}
		options.Height = n
	}
	if request.Query("q").String() != "" {
		options.Quality = request.Query("q").Int()
	}
	options.Download = request.Query("download").Bool()
	options.KeepAspectRatio = request.Query("crop").String() == ""
	if size := request.Query("size").String(); size != "" {
		parts := strings.Split(size, "x")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid size format %q: expected WxH", size)
		}
		w, err1 := strconv.Atoi(parts[0])
		h, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || w < 0 || h < 0 {
			return nil, fmt.Errorf("invalid size value %q: width and height must be non-negative integers", size)
		}
		options.Width = w
		options.Height = h
	}
	options.CropDirection = request.Query("dir").String()
	if options.Width > 0 && options.Height > 0 {
		options.KeepAspectRatio = false
	}
	// Accept both long form (format) and short alias (f).
	options.OutputFormat = queryFirst(request, "format", "f")
	if options.OutputFormat == "" {
		options.OutputFormat = t.Extension
	}

	// Parse video-specific options
	options.Preview = request.Query("preview").String()
	options.Thumbnail = request.Query("thumbnail").String()
	if request.Query("ss").String() != "" {
		options.SS = request.Query("ss").Int()
	}

	// Parse audio-specific options
	options.Detail = request.Query("detail").Bool()

	var ok bool
	if options.Encoder, ok = t.Encoders[options.OutputFormat]; !ok {
		return nil, fmt.Errorf("unsupported output format: %s", options.OutputFormat)
	}
	options.Encoder = t.Encoders[options.OutputFormat]

	if options.Width > 0 {
		options.Width = FindClosest(options.Width, ImageSizes)
	}

	if options.Height > 0 {
		options.Height = FindClosest(options.Height, ImageSizes)
	}

	if options.Quality > 0 {
		if options.Quality > 100 {
			return nil, fmt.Errorf("invalid quality value %d: must be between 1 and 100", options.Quality)
		}
		options.Quality = FindClosest(options.Quality, ImageQuality)
	}

	return options, nil
}

// FindClosest returns the largest value in sizes that is ≤ in.
// sizes must be sorted descending (largest first).
// Values larger than sizes[0] are clamped to sizes[0].
// Values smaller than sizes[len-1] are clamped to sizes[len-1].
func FindClosest(in int, sizes []int) int {
	if len(sizes) == 0 {
		return in
	}
	for _, size := range sizes {
		if in >= size {
			return size
		}
	}
	return sizes[len(sizes)-1]
}

type Encoder struct {
	Mime       string
	Parameters string
	Processor  func(input *Request) error
}

type Request struct {
	Domain            string
	Url               *evo.URL
	File              string
	Debug             bool
	TraceID           string
	Origin            *Origin
	Extension         string
	Request           *evo.Request
	Options           *Options
	MediaType         *Type
	Encoder           *Encoder
	OriginalFilePath  string
	StagedFilePath    string
	ProcessedFilePath string
	ProcessedMimeType string                 // MIME type of the processed file (e.g., for thumbnails)
	Metadata          map[string]interface{} `json:"metadata,omitempty"` // Metadata extracted from the file
}

// StageFile stages the file in a temp path for processing. it is necessary when a file is stored on a remote storage.
func (r *Request) StageFile() error {
	var err error
	var lastError error

	if r.Debug {
		log.Debug("Starting file staging", "trace_id", r.TraceID, "original_path", r.OriginalFilePath, "cache_dir", r.Origin.Project.CacheDir)
		r.Request.Set("X-Debug-Original-Path", r.OriginalFilePath)
		r.Request.Set("X-Debug-Cache-Dir", r.Origin.Project.CacheDir)
	}

	for i, storage := range r.Origin.Storages {
		if r.Debug {
			log.Debug("Trying storage", "trace_id", r.TraceID, "storage_index", i, "storage_type", storage.Type, "base_path", storage.BasePath)
			r.Request.Set(fmt.Sprintf("X-Debug-Storage-%d-Type", i), storage.Type)
			r.Request.Set(fmt.Sprintf("X-Debug-Storage-%d-BasePath", i), storage.BasePath)
		}

		r.StagedFilePath, err = storage.StageFile(r.OriginalFilePath, r.Origin.Project.CacheDir)
		if err == nil {
			if r.Debug {
				log.Debug("File staged successfully", "trace_id", r.TraceID, "storage_index", i, "staged_path", r.StagedFilePath)
				r.Request.Set("X-Debug-Storage-Success", fmt.Sprintf("storage-%d", i))
				r.Request.Set("X-Debug-Staged-Path", r.StagedFilePath)
			}
			return nil
		}

		lastError = err
		if r.Debug {
			log.Debug("Storage failed", "trace_id", r.TraceID, "storage_index", i, "error", err.Error())
			r.Request.Set(fmt.Sprintf("X-Debug-Storage-%d-Error", i), err.Error())
		}
	}

	if r.Debug {
		log.Debug("All storages failed", "trace_id", r.TraceID, "last_error", lastError.Error())
		r.Request.Set("X-Debug-Storage-Final-Error", lastError.Error())
	}

	return fmt.Errorf("failed to stage file: %v", lastError)
}

func (r *Request) ServeFile(mime string, filePath string) error {
	r.Request.Set("Content-Type", mime)
	file, err := os.Open(filePath)

	var c = r.Request.Context

	if err != nil {
		return fiber.ErrNotFound
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	fileSize := fi.Size()

	// Cache headers — use size+mtime as a lightweight ETag so browsers and
	// CDNs can revalidate without re-downloading the full file.
	etag := fmt.Sprintf(`"%x-%x"`, fi.ModTime().Unix(), fi.Size())
	lastMod := fi.ModTime().UTC().Format(time.RFC1123)
	c.Set("ETag", etag)
	c.Set("Last-Modified", lastMod)
	c.Set("Cache-Control", "public, max-age=86400")
	c.Set("Accept-Ranges", "bytes")

	// Conditional request: If-None-Match
	if c.Get("If-None-Match") == etag {
		c.Status(fiber.StatusNotModified)
		return nil
	}
	// Conditional request: If-Modified-Since
	if ims := c.Get("If-Modified-Since"); ims != "" {
		if t, err := time.Parse(time.RFC1123, ims); err == nil && !fi.ModTime().After(t) {
			c.Status(fiber.StatusNotModified)
			return nil
		}
	}

	rangeHeader := c.Get("Range")
	if rangeHeader == "" {
		c.Set("Content-Length", fmt.Sprintf("%d", fileSize))
		if r.Options.Download {
			c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filePath)))
		}
		c.Status(fiber.StatusOK)
		_, err := io.Copy(c, file)
		return err
	}

	// Parse the range header
	const bytesPrefix = "bytes="
	if !strings.HasPrefix(rangeHeader, bytesPrefix) {
		return fiber.ErrBadRequest
	}

	rangeHeader = strings.TrimPrefix(rangeHeader, bytesPrefix)

	// Handle multiple ranges (for now, we'll only serve the first range)
	// This is compliant with HTTP/1.1 spec which allows servers to ignore multipart ranges
	rangeSpecs := strings.Split(rangeHeader, ",")
	if len(rangeSpecs) == 0 {
		return fiber.ErrBadRequest
	}

	// Parse the first range specification
	rangeSpec := strings.TrimSpace(rangeSpecs[0])
	ranges := strings.Split(rangeSpec, "-")
	if len(ranges) != 2 {
		return fiber.ErrBadRequest
	}

	var start, end int64

	// Handle different range formats:
	// 1. "start-end" (e.g., "0-1023")
	// 2. "start-" (e.g., "1024-")
	// 3. "-suffix" (e.g., "-1024")
	if ranges[0] == "" && ranges[1] != "" {
		// Suffix-byte-range-spec: "-suffix"
		suffix, err := strconv.ParseInt(ranges[1], 10, 64)
		if err != nil || suffix <= 0 {
			return fiber.ErrBadRequest
		}
		if suffix >= fileSize {
			start = 0
		} else {
			start = fileSize - suffix
		}
		end = fileSize - 1
	} else if ranges[0] != "" && ranges[1] == "" {
		// Range from start to end of file: "start-"
		var err error
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil || start < 0 {
			return fiber.ErrBadRequest
		}
		if start >= fileSize {
			return fiber.ErrRequestedRangeNotSatisfiable
		}
		end = fileSize - 1
	} else if ranges[0] != "" && ranges[1] != "" {
		// Specific range: "start-end"
		var err error
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil || start < 0 {
			return fiber.ErrBadRequest
		}
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil || end < start {
			return fiber.ErrBadRequest
		}
		// Clamp end to file size
		if end >= fileSize {
			end = fileSize - 1
		}
		if start >= fileSize {
			return fiber.ErrRequestedRangeNotSatisfiable
		}
	} else {
		// Both empty: "-"
		return fiber.ErrBadRequest
	}

	length := end - start + 1
	if _, err = file.Seek(start, io.SeekStart); err != nil {
		return fiber.ErrInternalServerError
	}

	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Length", fmt.Sprintf("%d", length))
	c.Status(fiber.StatusPartialContent)
	_, err = io.CopyN(c, file, length)
	return err
}

type Project struct {
	ProjectID   int       `gorm:"column:project_id;primaryKey;autoIncrement" json:"project_id"`
	Name        string    `gorm:"column:name;size:255" json:"name"`
	Description string    `gorm:"column:description;size:255" json:"description"`
	Active      bool      `json:"column:active" json:"active"`
	CacheDir    string    `gorm:"column:cache_dir;size:255" json:"cache_dir"`
	CacheSize   string    `gorm:"column:cache_size;size:255" json:"cache_size"`
	CacheTTL    string    `gorm:"column:cache_ttl" json:"cache_ttl"`
	Storages    []Storage `gorm:"foreignKey:ProjectID"`
	Origins     []Origin  `gorm:"foreignKey:ProjectID"`
	types.CreatedAt
	types.UpdatedAt
	types.SoftDelete
}

func (Project) TableName() string {
	return "project"
}

type Storage struct {
	StorageID    int                  `gorm:"column:storage_id;primaryKey;autoIncrement" json:"storage_id"`
	ProjectID    int                  `gorm:"column:project_id;fk:project" json:"project_id"`
	Project      *Project             `gorm:"foreignKey:ProjectID;references:ProjectID"`
	Type         string               `gorm:"column:type;type:enum('fs','s3','ftp','sftp','http')" json:"type"`
	BasePath     string               `gorm:"column:base_path;size:255" json:"base_path"`
	ConfigString string               `gorm:"column:config_string;size:255" json:"config_string"`
	Priority     int                  `gorm:"column:priority" json:"priority"`
	FS           filesystem.Interface `gorm:"-"`
	types.CreatedAt
	types.UpdatedAt
	types.SoftDelete
	restify.API
}

func (Storage) TableName() string {
	return "storage"
}

func (s Storage) StageFile(path, cacheDir string) (string, error) {

	var filePath = filepath.Join(s.BasePath, path)
	var stagedPath = filepath.Join(cacheDir, path)

	// Guard against path traversal: the resolved paths must remain inside
	// their respective roots. filepath.Join cleans ".." sequences, so a
	// crafted path like "../../etc/passwd" would escape the base directory.
	// Only check when BasePath is set (S3/GCS storages have empty BasePath).
	if s.BasePath != "" {
		absBase := filepath.Clean(s.BasePath)
		if !strings.HasPrefix(filepath.Clean(filePath), absBase+string(filepath.Separator)) {
			return "", fmt.Errorf("path traversal detected: %q escapes storage root", path)
		}
	}
	absCache := filepath.Clean(cacheDir)
	if !strings.HasPrefix(filepath.Clean(stagedPath), absCache+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected: %q escapes cache root", path)
	}

	if gpath.IsFileExist(stagedPath) {
		return stagedPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(stagedPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Atomically acquire the lock using O_CREATE|O_EXCL — the kernel guarantees
	// that exactly one goroutine/process succeeds even under concurrent access,
	// eliminating the TOCTOU race of the previous Stat+Write approach.
	const lockTimeout = 5 * time.Minute
	const lockPollCycles = 10
	lockPath := stagedPath + ".lock"

	for c := 0; ; c++ {
		lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			// We own the lock.
			lf.Close()
			break
		}
		if !os.IsExist(err) {
			return stagedPath, fmt.Errorf("failed to create lock file: %w", err)
		}
		// Lock file already exists — check if it is stale.
		if info, statErr := os.Stat(lockPath); statErr == nil {
			if info.ModTime().Add(lockTimeout).Before(time.Now()) {
				// Stale lock left by a crashed writer — force-remove and retry immediately.
				os.Remove(lockPath)
				continue
			}
		}
		if c >= lockPollCycles {
			return STAGING, fmt.Errorf("file is locked")
		}
		time.Sleep(time.Second)
	}
	defer os.Remove(lockPath)
	// Download the file
	err = s.FS.StorageToDisk(filePath, stagedPath)
	if err != nil {
		return "", err
	}

	return stagedPath, nil
}

func (s *Storage) Init() {
	var err error
	s.BasePath = strings.Trim(s.BasePath, `\/`)
	switch s.Type {
	case "http":
		s.FS, err = httpfs.New(s.ConfigString)
		if err != nil {
			log.Error(err)
		}
	case "fs":
		s.FS, err = localfs.New(s.ConfigString)
		if err != nil {
			log.Error(err)
		}
	case "s3":
		s.FS, err = localS3.New(s.ConfigString)
		if err != nil {
			log.Error(err)
		}
	default:
		log.Panic("filesystem %s is not supported yet", s.Type)
	}

}

type Origin struct {
	OriginID   int        `gorm:"column:origin_id;primaryKey;autoIncrement" json:"origin_id"`
	ProjectID  int        `gorm:"column:project_id;fk:project" json:"project_id"`
	Project    *Project   `gorm:"foreignKey:ProjectID;references:ProjectID"`
	Domain     string     `gorm:"column:domain;size:255" json:"domain"`
	PrefixPath string     `gorm:"column:prefix_path;size:255" json:"prefix_path"`
	Storages   []*Storage `gorm:"-" json:"storages"`
	types.CreatedAt
	types.UpdatedAt
	types.SoftDelete
	restify.API
}

func (Origin) TableName() string {
	return "origin"
}

type VideoProfile struct {
	Profile string `gorm:"column:profile;size:255;primaryKey" json:"profile"`
	Width   int    `gorm:"column:width" json:"width"`
	Height  int    `gorm:"column:height" json:"height"`
	Quality int    `gorm:"column:quality" json:"quality"`
	Codec   string `gorm:"column:codec;size:255" json:"codec"`
	restify.API
}

func (VideoProfile) TableName() string {
	return "video_profile"
}

type Aspect struct {
	Name   string
	Width  float64
	Height float64
}

// List of common aspect ratios
var commonRatios = []Aspect{
	{"1:1", 1, 1},
	{"4:3", 4, 3},
	{"3:2", 3, 2},
	{"16:9", 16, 9},
	{"16:10", 16, 10},
	{"21:9", 21, 9},
	{"2:1", 2, 1},
	{"5:4", 5, 4},
	{"18:9", 18, 9},
	{"32:9", 32, 9},
}

func GetAspectRatioName(width, height float64) string {
	if width == 0 || height == 0 {
		return "Invalid"
	}
	inputRatio := width / height
	const tolerance = 0.02 // ~2% tolerance

	for _, aspect := range commonRatios {
		ratio := aspect.Width / aspect.Height
		if math.Abs(inputRatio-ratio) < tolerance {
			return aspect.Name
		}
	}
	return fmt.Sprintf("Custom (%.2f:1)", inputRatio)
}
