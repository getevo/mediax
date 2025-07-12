# Media Querying

MediaX provides powerful URL-based media processing. The general URL format is:

```
http://your-domain.com/path/to/media.ext?parameters
```

## Image Processing

### Basic Image Operations

```bash
# Original image
GET /images/photo.jpg

# Resize to specific dimensions
GET /images/photo.jpg?w=800&h=600

# Resize maintaining aspect ratio
GET /images/photo.jpg?w=800&ar=true

# Convert format
GET /images/photo.jpg?f=webp

# Adjust quality
GET /images/photo.jpg?q=85

# Combine operations
GET /images/photo.jpg?w=800&h=600&f=webp&q=90&crop=center
```

### Image Parameters

- `w` - Width in pixels
- `h` - Height in pixels  
- `f` - Output format (jpg, png, gif, webp, avif)
- `q` - Quality (1-100)
- `ar` - Keep aspect ratio (true/false)
- `crop` - Crop direction (center, top, bottom, left, right)

### Supported Image Formats

**Input**: JPG, PNG, GIF, WebP, AVIF
**Output**: JPG, PNG, GIF, WebP, AVIF

## Video Processing

### Basic Video Operations

```bash
# Original video
GET /videos/movie.mp4

# Generate thumbnail
GET /videos/movie.mp4?f=jpg

# Convert format
GET /videos/movie.mp4?f=webm

# Resize video
GET /videos/movie.mp4?w=1280&h=720

# Adjust quality
GET /videos/movie.mp4?q=75
```

### Video Parameters

- `w` - Width in pixels
- `h` - Height in pixels
- `f` - Output format (mp4, webm, avi, mov, mkv, flv, wmv, m4v, 3gp, ogv, jpg, png, webp, avif)
- `q` - Quality (1-100)
- `profile` - Video encoding profile
- `t` - Thumbnail timestamp (for thumbnail generation)

### Supported Video Formats

**Input**: MP4, WebM, AVI, MOV, MKV, FLV, WMV, M4V, 3GP, OGV
**Output**: MP4, WebM, AVI, MOV, MKV, FLV, WMV, M4V, 3GP, OGV
**Thumbnails**: JPG, PNG, WebP, AVIF

## Audio Processing

### Basic Audio Operations

```bash
# Original audio
GET /audio/song.mp3

# Convert format
GET /audio/song.mp3?f=flac

# Adjust quality
GET /audio/song.mp3?q=90

# Generate album art thumbnail
GET /audio/song.mp3?f=jpg

# Get metadata
GET /audio/song.mp3?detail=true
```

### Audio Parameters

- `f` - Output format (mp3, wav, flac, aac, ogg, m4a, wma, opus, jpg, png, webp, avif)
- `q` - Quality (1-100)
- `detail` - Return JSON metadata (true/false)

### Supported Audio Formats

**Input**: MP3, WAV, FLAC, AAC, OGG, M4A, WMA, Opus
**Output**: MP3, WAV, FLAC, AAC, OGG, M4A, WMA, Opus
**Album Art**: JPG, PNG, WebP, AVIF

## Document Processing

### Basic Document Operations

```bash
# Original document
GET /documents/report.pdf

# Generate thumbnail
GET /documents/report.pdf?thumbnail=800x600&f=jpg

# Generate thumbnail with different format
GET /documents/report.pdf?thumbnail=800x600&f=webp

# Generate thumbnail with quality setting
GET /documents/report.pdf?thumbnail=800x600&f=png&q=90
```

### Document Parameters

- `thumbnail` - Generate thumbnail with specified dimensions (e.g., 800x600, 1200x1700)
- `f` - Output format for thumbnails (jpg, png, webp, avif)
- `q` - Quality (1-100) for thumbnail generation

### Supported Document Formats

**Input**: 
- PDF
- Microsoft Office: DOCX, XLSX, PPTX
- Legacy Office: DOC, XLS, PPT
- OpenDocument: ODT, ODS, ODP
- Text: TXT, RTF, CSV
- Other: EPUB, XML

**Output**: Original format or thumbnail (JPG, PNG, WebP, AVIF)

## Advanced Features

### Debug Mode

Add `X-Debug: 1` header to get detailed processing information:

```bash
curl -H "X-Debug: 1" http://localhost:8080/images/photo.jpg?w=800
```

### Metadata Extraction

For audio files, use `detail=true` to get JSON metadata:

```bash
GET /audio/song.mp3?detail=true
```

Returns:
```json
{
  "title": "Song Title",
  "artist": "Artist Name",
  "album": "Album Name",
  "year": 2023,
  "duration": 180.5,
  "bitrate": 320,
  "format": "MP3"
}
```

## Processing Examples

### Image Processing Examples

```bash
# Create a 300x300 square thumbnail
GET /images/photo.jpg?w=300&h=300&crop=center

# Convert to WebP with 80% quality
GET /images/photo.jpg?f=webp&q=80

# Resize to max width 1200px keeping aspect ratio
GET /images/photo.jpg?w=1200&ar=true
```

### Video Processing Examples

```bash
# Create a thumbnail at 30 seconds
GET /videos/movie.mp4?f=jpg&t=30

# Convert to WebM with 720p resolution
GET /videos/movie.mp4?f=webm&w=1280&h=720

# Generate high-quality MP4
GET /videos/movie.mp4?f=mp4&q=90
```

### Audio Processing Examples

```bash
# Convert MP3 to high-quality FLAC
GET /audio/song.mp3?f=flac&q=100

# Extract album art as WebP
GET /audio/song.mp3?f=webp

# Get detailed metadata
GET /audio/song.mp3?detail=true
```

### Document Processing Examples

```bash
# Generate a PDF thumbnail at 1200x1700 resolution in WebP format
GET /documents/report.pdf?thumbnail=1200x1700&f=webp

# Create a high-quality PNG thumbnail of a Word document
GET /documents/proposal.docx?thumbnail=800x600&f=png&q=95

# Generate a thumbnail of a presentation with AVIF format
GET /documents/presentation.pptx?thumbnail=1024x768&f=avif
```
