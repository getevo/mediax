# Development and Extension

## Project Structure

```
mediax/
├── main.go                 # Application entry point
├── config.yml             # Configuration file
├── apps/
│   ├── mediax/            # Main application logic
│   │   ├── app.go         # App registration
│   │   ├── config.go      # Media type definitions
│   │   ├── controller.go  # Request handling
│   │   └── functions.go   # Utility functions
│   └── media/             # Media handling
│       ├── app.go         # Media app setup
│       └── media.go       # Core media structures
├── encoders/              # Media processors
│   ├── audio.go          # Audio processing
│   ├── image.go          # Image processing
│   ├── video.go          # Video processing
│   └── mediatype.go      # Media type utilities
└── docs/                 # Documentation
```

## Adding New Media Types

1. **Define the media type** in `apps/mediax/config.go`:

```text
"pdf": {
    Extension: "pdf",
    Mime:      "application/pdf",
    Encoders:  map[string]*media.Encoder{"jpg": &encoders.PdfToImage},
},
```

2. **Create the encoder** in `encoders/`:

```text
var PdfToImage = media.Encoder{
    Mime:      "image/jpeg",
    Processor: ProcessPdf,
}

var ProcessPdf = func(input *media.Request) error {
    // Implementation here
    return nil
}
```

## Adding New Storage Backends

1. **Implement the storage interface** in `apps/media/media.go`
2. **Add configuration support** in the Storage struct
3. **Update the Init() method** to handle the new storage type

### Example: Adding Redis Storage

```text
// In apps/media/media.go
func (s *Storage) Init() error {
    switch s.Type {
    case "redis":
        // Initialize Redis client
        s.FileSystem = &RedisStorage{
            Client: redis.NewClient(&redis.Options{
                Addr: s.Server,
                Password: s.Password,
            }),
        }
    // ... other cases
    }
}
```

## Custom Encoders

Create custom encoders by implementing the `media.Encoder` interface:

```text
type Encoder struct {
    Mime      string
    Processor func(*Request) error
}
```

### Example: Custom Image Watermark Encoder

```text
var WatermarkEncoder = media.Encoder{
    Mime:      "image/jpeg",
    Processor: ProcessWatermark,
}

var ProcessWatermark = func(input *media.Request) error {
    // Add watermark to image
    args := []string{
        input.StagedFilePath,
        "watermark.png",
        "-gravity", "southeast",
        "-composite",
        input.ProcessedFilePath,
    }

    cmd := exec.Command("convert", args...)
    return cmd.Run()
}
```

## Extending Processing Options

Add new processing options in `apps/media/media.go`:

```text
type Options struct {
    Width           int    `json:"width"`
    Height          int    `json:"height"`
    Quality         int    `json:"quality"`
    OutputFormat    string `json:"output_format"`
    // Add your custom options here
    Watermark       string `json:"watermark"`
    Brightness      int    `json:"brightness"`
    Contrast        int    `json:"contrast"`
}
```

## Database Models

The system uses GORM for database operations. Main models:

### Project Model
```text
type Project struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"size:255;not null"`
    Description string    `gorm:"type:text"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time `gorm:"index"`
}
```

### Origin Model
```text
type Origin struct {
    ID         uint      `gorm:"primaryKey"`
    ProjectID  uint      `gorm:"not null"`
    Domain     string    `gorm:"size:255;not null;uniqueIndex"`
    PrefixPath string    `gorm:"size:255"`
    Project    Project   `gorm:"foreignKey:ProjectID"`
    Storages   []*Storage `gorm:"-"`
}
```

### Storage Model
```text
type Storage struct {
    ID        uint   `gorm:"primaryKey"`
    ProjectID uint   `gorm:"not null"`
    Type      string `gorm:"size:50;not null"`
    Priority  int    `gorm:"default:1"`
    // Storage-specific fields
    Path      string `gorm:"size:500"`
    Bucket    string `gorm:"size:255"`
    Region    string `gorm:"size:100"`
}
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./apps/media -v

# Run specific test
go test -run TestImageProcessing ./encoders
```

### Integration Tests

```bash
# Run integration tests with database
go test -tags=integration ./...
```

### Example Test

```text
func TestImageResize(t *testing.T) {
    req := &media.Request{
        StagedFilePath: "test_image.jpg",
        Options: media.Options{
            Width:  800,
            Height: 600,
            OutputFormat: "jpg",
        },
    }

    err := encoders.Imagick(req)
    assert.NoError(t, err)
    assert.FileExists(t, req.ProcessedFilePath)
}
```

## Performance Optimization

### Caching Strategies

1. **File-based caching**: Processed files are cached on disk
2. **Memory caching**: Frequently accessed metadata in memory
3. **CDN integration**: Use CDN for global distribution

### Optimization Tips

```text
// Use goroutines for parallel processing
func ProcessMultipleFiles(files []string) {
    var wg sync.WaitGroup
    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()
            // Process file
        }(file)
    }
    wg.Wait()
}
```

## Debugging

### Enable Debug Mode

```text
// In controller
if request.Header("X-Debug") == "1" {
    log.Debug("Processing request", "file", req.OriginalFilePath)
}
```

### Logging Best Practices

```text
import "github.com/getevo/evo/v2/lib/log"

// Use structured logging
log.Info("File processed", 
    "file", filename,
    "size", fileSize,
    "duration", processingTime,
)

// Log errors with context
log.Error("Processing failed",
    "file", filename,
    "error", err.Error(),
    "trace_id", traceID,
)
```

## Contributing Guidelines

### Code Style

1. Follow Go conventions and use `gofmt`
2. Write comprehensive tests for new features
3. Document public functions and types
4. Use meaningful variable and function names

### Pull Request Process

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes with tests
4. Run tests: `go test ./...`
5. Submit a pull request with detailed description

### Commit Message Format

```
type(scope): description

- feat: new feature
- fix: bug fix
- docs: documentation changes
- test: adding tests
- refactor: code refactoring
```

## Advanced Topics

### Custom Middleware

```text
func AuthMiddleware(c *fiber.Ctx) error {
    token := c.Get("Authorization")
    if !isValidToken(token) {
        return c.Status(401).JSON(fiber.Map{
            "error": "Unauthorized",
        })
    }
    return c.Next()
}
```

### Plugin System

```text
type Plugin interface {
    Name() string
    Process(*media.Request) error
}

func RegisterPlugin(p Plugin) {
    plugins[p.Name()] = p
}
```
