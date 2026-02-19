# MediaX - Full Feature Media Serving Server

MediaX is a powerful, scalable media processing and serving server built with Go. It provides on-the-fly media transformation, multiple storage backend support, and high-performance media delivery for images, videos, audio files, and documents.

## Features

- **Multi-format Support**: Images (JPG, PNG, GIF, WebP, AVIF), Videos (MP4, WebM, AVI, MOV, MKV, FLV, WMV, M4V, 3GP, OGV), Audio (MP3, WAV, FLAC, AAC, OGG, M4A, WMA, Opus), Documents (PDF, DOCX, XLSX, PPTX, DOC, XLS, PPT, ODT, ODS, ODP, TXT, RTF, CSV, EPUB, XML)
- **On-the-fly Processing**: Real-time image resizing, video transcoding, audio conversion, and thumbnail generation
- **Multiple Storage Backends**: Local filesystem, AWS S3, and HTTP-based storage
- **Domain-based Configuration**: Multi-tenant support with domain-specific settings
- **High Performance**: Built on the EVO framework with efficient caching and processing
- **RESTful API**: Clean API for media management and processing
- **Docker Support**: Easy deployment with Docker containers

## Quick Start

```bash
# Clone the repository
git clone https://github.com/getevo/mediax
cd mediax

# Install dependencies
go mod download

# Run the application
go run main.go
```

The server will start on `http://localhost:8080` by default.

## Documentation

Comprehensive documentation is available in the `docs/` directory:

### 📚 Core Documentation
- **[Setup and Requirements](docs/setup.md)** - Installation, dependencies, and database setup
- **[Configuration](docs/configuration.md)** - Configuration options and how to run MediaX
- **[Storage Setup](docs/storage.md)** - Configure local, S3, and HTTP storage backends

### 🎯 Usage Guides  
- **[Media Querying](docs/media-querying.md)** - Complete guide to processing images, videos, and audio
- **[API Reference](docs/api-reference.md)** - REST API documentation and SDK examples

### 🚀 Deployment & Operations
- **[Docker Deployment](docs/docker.md)** - Docker, Docker Compose, and Kubernetes deployment
- **[Performance Tuning](docs/performance.md)** - Optimization, scaling, and monitoring
- **[Security](docs/security.md)** - Security best practices and configuration

### 🛠️ Development
- **[Development Guide](docs/development.md)** - Extending MediaX, adding features, and contributing
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and debugging guide

## Example Usage

### Basic Image Processing
```bash
# Resize image to 800x600
GET /images/photo.jpg?w=800&h=600

# Convert to WebP with 90% quality
GET /images/photo.jpg?f=webp&q=90

# Generate thumbnail maintaining aspect ratio
GET /images/photo.jpg?w=300&ar=true
```

### Video Processing
```bash
# Generate video thumbnail
GET /videos/movie.mp4?f=jpg&t=30

# Convert to WebM format
GET /videos/movie.mp4?f=webm&w=1280&h=720
```

### Audio Processing
```bash
# Convert audio format
GET /audio/song.mp3?f=flac&q=100

# Extract album art
GET /audio/song.mp3?f=jpg

# Get metadata
GET /audio/song.mp3?detail=true
```

### Document Processing
```bash
# Generate thumbnail from PDF
GET /documents/document.pdf?f=jpg&thumbnail=480p

# Generate thumbnail from Office document
GET /documents/presentation.pptx?f=png&thumbnail=1080p

# Get document metadata
GET /documents/spreadsheet.xlsx?detail=true
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[Add your license information here]

## Support

[Add support contact information here]
