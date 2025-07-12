# Storage Setup

MediaX supports multiple storage backends that can be configured per project:

## Local Filesystem Storage

```yaml
# Example storage configuration in database
Type: "local"
Path: "/var/media/storage"
Priority: 1
```

## AWS S3 Storage

```yaml
# Example S3 storage configuration
Type: "s3"
Bucket: "my-media-bucket"
Region: "us-west-2"
AccessKey: "your-access-key"
SecretKey: "your-secret-key"
Priority: 2
```

## HTTP Storage

```yaml
# Example HTTP storage configuration
Type: "http"
BaseURL: "https://cdn.example.com/media/"
Priority: 3
```

## Storage Priority

Storages are tried in order of priority (lowest number first). If a file is not found in the primary storage, the system will try the next storage backend.

## Configuration Examples

### Multiple Storage Backends

You can configure multiple storage backends for redundancy and performance:

1. **Primary**: Local filesystem for fast access
2. **Secondary**: S3 for backup and scalability
3. **Tertiary**: HTTP CDN for global distribution

### Storage Failover

The system automatically tries storage backends in priority order:
- If the primary storage fails, it tries the secondary
- If the secondary fails, it tries the tertiary
- This ensures high availability of media files

### Best Practices

- Use local storage for frequently accessed files
- Use S3 for long-term storage and backup
- Use HTTP storage for CDN integration
- Set appropriate priorities based on performance requirements
- Monitor storage health and availability
