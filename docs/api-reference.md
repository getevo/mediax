# API Reference

## Admin API

The admin API is available at `/admin` prefix and provides endpoints for managing:

- Projects
- Origins (domains)
- Storage configurations
- Video profiles

### Authentication

All admin API endpoints require authentication. Include the authorization header:

```
Authorization: Bearer <your-token>
```

### Projects API

#### List Projects
```
GET /admin/projects
```

Response:
```json
{
  "data": [
    {
      "id": 1,
      "name": "My Project",
      "description": "Project description",
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ]
}
```

#### Create Project
```
POST /admin/projects
Content-Type: application/json

{
  "name": "New Project",
  "description": "Project description"
}
```

#### Update Project
```
PUT /admin/projects/{id}
Content-Type: application/json

{
  "name": "Updated Project",
  "description": "Updated description"
}
```

#### Delete Project
```
DELETE /admin/projects/{id}
```

### Origins API

#### List Origins
```
GET /admin/origins
```

Response:
```json
{
  "data": [
    {
      "id": 1,
      "project_id": 1,
      "domain": "example.com",
      "prefix_path": "/media",
      "project": {
        "id": 1,
        "name": "My Project"
      }
    }
  ]
}
```

#### Create Origin
```
POST /admin/origins
Content-Type: application/json

{
  "project_id": 1,
  "domain": "example.com",
  "prefix_path": "/media"
}
```

#### Update Origin
```
PUT /admin/origins/{id}
Content-Type: application/json

{
  "domain": "updated-example.com",
  "prefix_path": "/assets"
}
```

#### Delete Origin
```
DELETE /admin/origins/{id}
```

### Storage API

#### List Storages
```
GET /admin/storages
```

Response:
```json
{
  "data": [
    {
      "id": 1,
      "project_id": 1,
      "type": "local",
      "priority": 1,
      "path": "/var/media/storage"
    },
    {
      "id": 2,
      "project_id": 1,
      "type": "s3",
      "priority": 2,
      "bucket": "my-media-bucket",
      "region": "us-west-2"
    }
  ]
}
```

#### Create Storage
```
POST /admin/storages
Content-Type: application/json

{
  "project_id": 1,
  "type": "local",
  "priority": 1,
  "path": "/var/media/storage"
}
```

#### Create S3 Storage
```
POST /admin/storages
Content-Type: application/json

{
  "project_id": 1,
  "type": "s3",
  "priority": 2,
  "bucket": "my-media-bucket",
  "region": "us-west-2",
  "access_key": "your-access-key",
  "secret_key": "your-secret-key"
}
```

#### Update Storage
```
PUT /admin/storages/{id}
Content-Type: application/json

{
  "priority": 3,
  "path": "/new/media/path"
}
```

#### Delete Storage
```
DELETE /admin/storages/{id}
```

### Video Profiles API

#### List Video Profiles
```
GET /admin/video-profiles
```

Response:
```json
{
  "data": [
    {
      "id": 1,
      "profile": "hd",
      "width": 1280,
      "height": 720,
      "bitrate": "2000k",
      "codec": "h264"
    }
  ]
}
```

#### Create Video Profile
```
POST /admin/video-profiles
Content-Type: application/json

{
  "profile": "4k",
  "width": 3840,
  "height": 2160,
  "bitrate": "8000k",
  "codec": "h264"
}
```

#### Update Video Profile
```
PUT /admin/video-profiles/{id}
Content-Type: application/json

{
  "bitrate": "10000k",
  "codec": "h265"
}
```

#### Delete Video Profile
```
DELETE /admin/video-profiles/{id}
```

## Media Serving API

All media requests are handled through the main domain routing:

```
GET /{path-to-media}?{processing-parameters}
```

### Common Parameters

- `w` - Width in pixels
- `h` - Height in pixels
- `f` - Output format
- `q` - Quality (1-100)

### Response Headers

MediaX sets appropriate headers for caching and content delivery:

- `Content-Type`: Correct MIME type for processed media
- `Cache-Control`: Caching directives
- `X-Trace-ID`: Request tracing ID (in debug mode)
- `X-Debug-*`: Debug information (when debug mode is enabled)

### Error Responses

#### 400 Bad Request
```json
{
  "error": "Invalid parameters",
  "message": "Width must be a positive integer"
}
```

#### 403 Forbidden
```json
{
  "error": "Forbidden domain",
  "message": "Domain not configured"
}
```

#### 404 Not Found
```json
{
  "error": "File not found",
  "message": "The requested file does not exist"
}
```

#### 415 Unsupported Media Type
```json
{
  "error": "Unsupported media type",
  "message": "File format not supported"
}
```

#### 500 Internal Server Error
```json
{
  "error": "Processing failed",
  "message": "Unable to process media file"
}
```

### Debug Mode

Enable debug mode by adding the `X-Debug: 1` header to requests:

```bash
curl -H "X-Debug: 1" http://localhost:8080/images/photo.jpg?w=800
```

Debug response headers:
- `X-Debug-Host`: Request host
- `X-Debug-Extension`: Detected file extension
- `X-Debug-MediaType`: Media type configuration
- `X-Debug-Options`: Processing options
- `X-Debug-Error`: Error details (if any)

### Rate Limiting

The API implements rate limiting to prevent abuse:

- **Default**: 100 requests per minute per IP
- **Burst**: Up to 200 requests in a short period
- **Headers**: Rate limit information in response headers

Rate limit headers:
- `X-RateLimit-Limit`: Requests allowed per window
- `X-RateLimit-Remaining`: Requests remaining in current window
- `X-RateLimit-Reset`: Time when the rate limit resets

### Caching

MediaX implements intelligent caching:

- **Processed files**: Cached on disk for fast subsequent access
- **Cache headers**: Appropriate cache-control headers set
- **ETags**: Entity tags for efficient cache validation
- **CDN friendly**: Optimized for CDN integration

Cache-related headers:
- `Cache-Control`: Caching directives
- `ETag`: Entity tag for cache validation
- `Last-Modified`: Last modification time
- `Expires`: Expiration time

### Content Delivery

- **Streaming**: Large files are streamed for efficient delivery
- **Compression**: Automatic compression for supported formats
- **Range requests**: Support for partial content requests
- **CORS**: Cross-origin resource sharing support

## SDK and Client Libraries

### JavaScript/Node.js

```javascript
const MediaX = require('mediax-client');

const client = new MediaX({
  baseURL: 'https://your-domain.com',
  apiKey: 'your-api-key'
});

// Process image
const imageUrl = client.image('/path/to/image.jpg')
  .width(800)
  .height(600)
  .format('webp')
  .quality(90)
  .url();

// Process video
const videoUrl = client.video('/path/to/video.mp4')
  .width(1280)
  .height(720)
  .format('webm')
  .url();
```

### Python

```python
from mediax import MediaXClient

client = MediaXClient(
    base_url='https://your-domain.com',
    api_key='your-api-key'
)

# Process image
image_url = (client.image('/path/to/image.jpg')
    .width(800)
    .height(600)
    .format('webp')
    .quality(90)
    .url())

# Process video
video_url = (client.video('/path/to/video.mp4')
    .width(1280)
    .height(720)
    .format('webm')
    .url())
```

### PHP

```php
<?php
use MediaX\Client;

$client = new Client([
    'base_url' => 'https://your-domain.com',
    'api_key' => 'your-api-key'
]);

// Process image
$imageUrl = $client->image('/path/to/image.jpg')
    ->width(800)
    ->height(600)
    ->format('webp')
    ->quality(90)
    ->url();

// Process video
$videoUrl = $client->video('/path/to/video.mp4')
    ->width(1280)
    ->height(720)
    ->format('webm')
    ->url();
?>
```