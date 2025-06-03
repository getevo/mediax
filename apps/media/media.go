package media

import (
	"fmt"
	httpfs "mediax/http"
	"mediax/s3"

	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/db/types"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/restify"
	"github.com/gofiber/fiber/v2"
	"io"
	"mediax/filesystem"
	"mediax/localfs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
}

func (o Options) ToString() string {
	return fmt.Sprintf("%dx%da%tq%dd%sp%s", o.Width, o.Height, o.KeepAspectRatio, o.Quality, o.CropDirection, o.Profile)
}

func (t *Type) ParseOptions(request *evo.Request) (*Options, error) {
	options := &Options{}
	if request.Query("width").String() != "" {
		options.Width = request.Query("width").Int()
	}
	if request.Query("height").String() != "" {
		options.Height = request.Query("height").Int()
	}
	if request.Query("q").String() != "" {
		options.Quality = request.Query("q").Int()
	}
	options.Download = request.Query("download").Bool()
	options.KeepAspectRatio = request.Query("crop").String() == ""
	if size := request.Query("size").String(); size != "" {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			options.Width, _ = strconv.Atoi(parts[0])
			options.Height, _ = strconv.Atoi(parts[1])
		}
	}
	options.CropDirection = request.Query("dir").String()
	if options.Width > 0 && options.Height > 0 {
		options.KeepAspectRatio = false
	}
	options.OutputFormat = request.Query("format").String()
	if options.OutputFormat == "" {
		options.OutputFormat = t.Extension
	}

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
		options.Quality = FindClosest(options.Quality, ImageQuality)
	}

	return options, nil
}

func FindClosest(in int, sizes []int) int {
	var x int
	for _, size := range sizes {
		x = size
		if in > size {
			if x == 0 {
				return size
			}
			return x
		}
	}
	return in
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
	Origin            *Origin
	Extension         string
	Request           *evo.Request
	Options           *Options
	MediaType         *Type
	Encoder           *Encoder
	OriginalFilePath  string
	StagedFilePath    string
	ProcessedFilePath string
}

// StageFile stages the file in a temp path for processing. it is necessary when a file is stored on a remote storage.
func (r *Request) StageFile() error {
	var err error
	for _, storage := range r.Origin.Storages {
		r.StagedFilePath, err = storage.StageFile(r.OriginalFilePath, r.Origin.Project.CacheDir)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to stage file: %v", err)
}

func (r *Request) ServeFile(mime string, filePath string) error {
	fmt.Println("Serving file:", filePath, mime)
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
	ranges := strings.Split(rangeHeader, "-")
	if len(ranges) != 2 {
		return fiber.ErrBadRequest
	}

	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil || start < 0 {
		return fiber.ErrBadRequest
	}

	var end int64
	if ranges[1] != "" {
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil || end < start {
			return fiber.ErrBadRequest
		}
	} else {
		end = fileSize - 1
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
	if !gpath.IsFileExist(stagedPath) {
		err := s.FS.StorageToDisk(filePath, stagedPath)
		if err != nil {
			return "", err
		}
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
		s.FS, err = s3.New(s.ConfigString)
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
