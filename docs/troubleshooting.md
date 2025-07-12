# Troubleshooting

## Common Issues

### 1. FFmpeg Not Found

**Problem**: Error message "ffmpeg: command not found" or video/audio processing fails.

**Solutions**:

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Alpine Linux (Docker)
apk add ffmpeg

# Verify installation
ffmpeg -version
```

**Docker Solution**:
```dockerfile
FROM alpine:latest
RUN apk add --no-cache ffmpeg
```

### 2. ImageMagick Not Found

**Problem**: Error message "convert: command not found" or image processing fails.

**Solutions**:

```bash
# Ubuntu/Debian
sudo apt install imagemagick

# macOS
brew install imagemagick

# Alpine Linux (Docker)
apk add imagemagick

# Verify installation
convert -version
```

**Policy Issues**:
```bash
# Check ImageMagick policy
cat /etc/ImageMagick-6/policy.xml

# Common fix for security policy restrictions
sudo sed -i 's/rights="none" pattern="PDF"/rights="read|write" pattern="PDF"/' /etc/ImageMagick-6/policy.xml
```

### 3. Database Connection Failed

**Problem**: Cannot connect to database, connection refused errors.

**Diagnosis**:
```bash
# Check database service status
systemctl status mysql
docker-compose ps mysql

# Test connection manually
mysql -h localhost -u mediax_user -p mediax

# Check network connectivity
telnet localhost 3306
```

**Solutions**:

```yaml
# Correct database configuration
Database:
  Type: mysql
  Server: localhost:3306  # or mysql:3306 in Docker
  Database: "mediax"
  Username: mediax_user
  Password: "correct_password"
  Params: "parseTime=true"
```

**Docker Network Issues**:
```bash
# Check Docker networks
docker network ls
docker network inspect mediax_default

# Recreate network
docker-compose down
docker-compose up -d
```

### 4. File Not Found

**Problem**: 404 errors when accessing media files.

**Diagnosis**:
```bash
# Check file existence
ls -la /var/media/path/to/file.jpg

# Check permissions
ls -la /var/media/
stat /var/media/path/to/file.jpg

# Check storage configuration
curl -H "X-Debug: 1" http://localhost:8080/path/to/file.jpg
```

**Solutions**:

1. **Verify Storage Configuration**:
```sql
SELECT * FROM storages WHERE project_id = 1;
```

2. **Check File Permissions**:
```bash
# Fix permissions
chown -R mediax:mediax /var/media
chmod -R 755 /var/media
```

3. **Verify Domain Configuration**:
```sql
SELECT * FROM origins WHERE domain = 'your-domain.com';
```

### 5. Processing Timeout

**Problem**: Large files fail to process with timeout errors.

**Solutions**:

```yaml
# Increase timeouts in config.yml
HTTP:
  ReadTimeout: 60s
  WriteTimeout: 120s
```

```nginx
# Nginx timeout configuration
proxy_read_timeout 300s;
proxy_send_timeout 300s;
client_body_timeout 300s;
```

**Docker Configuration**:
```yaml
services:
  mediax:
    environment:
      - PROCESSING_TIMEOUT=300s
    deploy:
      resources:
        limits:
          memory: 4G
```

### 6. High Memory Usage

**Problem**: Application consuming too much memory, OOM kills.

**Diagnosis**:
```bash
# Check memory usage
docker stats mediax
top -p $(pgrep mediax)

# Go memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Solutions**:

1. **Optimize ImageMagick**:
```bash
export MAGICK_MEMORY_LIMIT=1GB
export MAGICK_MAP_LIMIT=2GB
export MAGICK_DISK_LIMIT=4GB
```

2. **Go Runtime Tuning**:
```bash
export GOGC=50          # More aggressive GC
export GOMEMLIMIT=2GiB  # Memory limit
```

3. **Process Large Files in Chunks**:
```text
// Implement streaming for large files
func processLargeFile(input *media.Request) error {
    if fileSize > maxMemorySize {
        return processWithTempFile(input)
    }
    return processInMemory(input)
}
```

### 7. Slow Performance

**Problem**: Media processing is slow, high response times.

**Diagnosis**:
```bash
# Check CPU usage
top -p $(pgrep mediax)

# Check I/O wait
iostat -x 1

# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile
```

**Solutions**:

1. **Enable Caching**:
```yaml
Cache:
  Enabled: true
  Directory: "/var/cache/mediax"
  MaxSize: "10GB"
```

2. **Optimize Storage**:
```bash
# Use SSD for cache
mount /dev/nvme0n1 /var/cache/mediax

# Optimize mount options
mount -o noatime,nodiratime /dev/sdb1 /var/media
```

3. **Parallel Processing**:
```text
// Use worker pools
pool := NewWorkerPool(runtime.NumCPU())
pool.Start()
```

### 8. Docker Issues

**Problem**: Container fails to start or crashes.

**Diagnosis**:
```bash
# Check container logs
docker-compose logs mediax

# Check container status
docker-compose ps

# Inspect container
docker inspect mediax_container
```

**Common Solutions**:

1. **Permission Issues**:
```dockerfile
# Fix in Dockerfile
RUN addgroup -g 1001 -S mediax \
    && adduser -u 1001 -S mediax -G mediax
USER mediax
```

2. **Resource Limits**:
```yaml
services:
  mediax:
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.0'
```

3. **Volume Mounting**:
```yaml
volumes:
  - ./media:/var/media:rw
  - ./cache:/var/cache:rw
```

### 9. SSL/TLS Issues

**Problem**: HTTPS not working, certificate errors.

**Diagnosis**:
```bash
# Test SSL configuration
openssl s_client -connect your-domain.com:443

# Check certificate
curl -vI https://your-domain.com

# Verify certificate files
openssl x509 -in cert.pem -text -noout
```

**Solutions**:

1. **Let's Encrypt Setup**:
```bash
certbot --nginx -d your-domain.com
systemctl enable certbot.timer
```

2. **Nginx SSL Configuration**:
```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
ssl_prefer_server_ciphers off;
```

### 10. Rate Limiting Issues

**Problem**: Legitimate requests being blocked by rate limiting.

**Diagnosis**:
```bash
# Check rate limit logs
grep "rate limit" /var/log/nginx/error.log

# Test rate limits
for i in {1..20}; do curl http://localhost:8080/test.jpg; done
```

**Solutions**:

1. **Adjust Rate Limits**:
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=20r/s;
limit_req zone=api burst=50 nodelay;
```

2. **Whitelist IPs**:
```nginx
geo $limit {
    default 1;
    10.0.0.0/8 0;      # Internal network
    192.168.0.0/16 0;  # Private network
}

map $limit $limit_key {
    0 "";
    1 $binary_remote_addr;
}

limit_req_zone $limit_key zone=api:10m rate=10r/s;
```

## Debug Mode

### Enabling Debug Mode

Add the `X-Debug: 1` header to requests for detailed information:

```bash
curl -H "X-Debug: 1" http://localhost:8080/images/photo.jpg?w=800
```

### Debug Response Headers

- `X-Debug-Host`: Request host
- `X-Debug-Extension`: Detected file extension
- `X-Debug-MediaType`: Media type configuration
- `X-Debug-Options`: Processing options
- `X-Debug-Error`: Error details (if any)

### Application Logging

```text
// Enable debug logging
log.SetLevel(log.DebugLevel)

// Add trace IDs to logs
func processRequest(req *media.Request) {
    log.Debug("Processing started", 
        "trace_id", req.TraceID,
        "file", req.OriginalFilePath,
        "options", req.Options)
}
```

## Health Checks

### Application Health Check

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health check
curl http://localhost:8080/health?detailed=true
```

### Database Health Check

```bash
# Check database connectivity
mysql -h localhost -u mediax_user -p -e "SELECT 1"

# Check table status
mysql -h localhost -u mediax_user -p mediax -e "SHOW TABLE STATUS"
```

### Storage Health Check

```bash
# Check local storage
df -h /var/media
ls -la /var/media

# Check S3 connectivity
aws s3 ls s3://your-bucket/

# Check HTTP storage
curl -I https://cdn.example.com/test-file.jpg
```

## Log Analysis

### Common Log Patterns

```bash
# Find errors
grep -i error /var/log/mediax/*.log

# Find slow requests
grep "processing_time.*[5-9][0-9][0-9][0-9]" /var/log/mediax/access.log

# Find failed requests
grep "status.*[45][0-9][0-9]" /var/log/mediax/access.log

# Memory issues
grep -i "out of memory\|oom" /var/log/mediax/*.log
```

### Log Rotation

```bash
# Configure logrotate
cat > /etc/logrotate.d/mediax << EOF
/var/log/mediax/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload mediax
    endscript
}
EOF
```

## Performance Monitoring

### Real-time Monitoring

```bash
# Monitor requests
watch -n 1 'curl -s http://localhost:8080/metrics | grep mediax_requests_total'

# Monitor memory
watch -n 1 'docker stats mediax --no-stream'

# Monitor disk I/O
iostat -x 1
```

### Prometheus Queries

```promql
# Request rate
rate(mediax_requests_total[5m])

# Error rate
rate(mediax_requests_total{status=~"5.."}[5m]) / rate(mediax_requests_total[5m])

# Processing time percentiles
histogram_quantile(0.95, rate(mediax_processing_duration_seconds_bucket[5m]))
```

## Recovery Procedures

### Database Recovery

```bash
# Restore from backup
mysql -u root -p mediax < backup.sql

# Repair corrupted tables
mysqlcheck --repair --all-databases
```

### File System Recovery

```bash
# Check filesystem
fsck /dev/sdb1

# Restore from backup
rsync -av /backup/media/ /var/media/
```

### Container Recovery

```bash
# Restart containers
docker-compose restart mediax

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## Getting Help

### Collecting Debug Information

```bash
#!/bin/bash
# debug-info.sh - Collect system information

echo "=== MediaX Debug Information ==="
echo "Date: $(date)"
echo "Hostname: $(hostname)"
echo

echo "=== Application Status ==="
docker-compose ps
echo

echo "=== Resource Usage ==="
docker stats --no-stream
echo

echo "=== Recent Logs ==="
docker-compose logs --tail=50 mediax
echo

echo "=== Configuration ==="
cat config.yml
echo

echo "=== Database Status ==="
mysql -h localhost -u mediax_user -p -e "SHOW PROCESSLIST; SHOW STATUS LIKE 'Threads%';"
```

### Support Channels

1. **GitHub Issues**: Report bugs and feature requests
2. **Documentation**: Check the docs directory
3. **Community Forum**: Ask questions and share solutions
4. **Professional Support**: Contact for enterprise support

### Before Reporting Issues

1. Check this troubleshooting guide
2. Search existing GitHub issues
3. Collect debug information
4. Provide minimal reproduction steps
5. Include system information and logs