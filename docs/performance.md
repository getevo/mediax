# Performance Tuning

## Caching Strategy

### File-based Caching

MediaX automatically caches processed files on disk to avoid reprocessing:

```yaml
# Configuration for cache settings
Cache:
  Directory: "/var/cache/mediax"
  MaxSize: "10GB"
  TTL: "24h"
  CleanupInterval: "1h"
```

### Memory Caching

Implement memory caching for frequently accessed metadata:

```text
// Example memory cache configuration
type MemoryCache struct {
    MaxSize     int
    TTL         time.Duration
    CleanupTime time.Duration
}

var cache = &MemoryCache{
    MaxSize:     1000,
    TTL:         time.Hour,
    CleanupTime: time.Minute * 10,
}
```

### CDN Integration

Configure CDN for global distribution:

```nginx
# Nginx configuration for CDN
location ~* \.(jpg|jpeg|png|gif|webp|avif|mp4|webm|mp3|flac)$ {
    proxy_pass http://mediax;
    proxy_cache mediax_cache;
    proxy_cache_valid 200 1d;
    proxy_cache_key $uri$is_args$args;
    
    # CDN headers
    add_header X-Cache-Status $upstream_cache_status;
    add_header Cache-Control "public, max-age=86400";
}
```

## Scaling Strategies

### Horizontal Scaling

#### Load Balancer Configuration

```nginx
upstream mediax_backend {
    least_conn;
    server mediax1:8080 weight=3;
    server mediax2:8080 weight=3;
    server mediax3:8080 weight=2;
    
    # Health checks
    keepalive 32;
}

server {
    location / {
        proxy_pass http://mediax_backend;
        proxy_next_upstream error timeout invalid_header http_500;
        proxy_connect_timeout 2s;
        proxy_read_timeout 30s;
    }
}
```

#### Docker Swarm Deployment

```yaml
version: '3.8'
services:
  mediax:
    image: mediax:latest
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    networks:
      - mediax-network
```

#### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mediax
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mediax
  template:
    metadata:
      labels:
        app: mediax
    spec:
      containers:
      - name: mediax
        image: mediax:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
---
apiVersion: v1
kind: Service
metadata:
  name: mediax-service
spec:
  selector:
    app: mediax
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Vertical Scaling

#### Resource Optimization

```yaml
# Docker Compose resource limits
services:
  mediax:
    image: mediax:latest
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

#### Go Runtime Optimization

```bash
# Environment variables for Go runtime
export GOGC=100                    # Garbage collection target
export GOMAXPROCS=4               # Maximum number of CPUs
export GOMEMLIMIT=3GiB            # Memory limit
```

## Database Optimization

### Connection Pooling

```yaml
Database:
  MaxOpenConns: 100
  MaxIdleConns: 10
  ConnMaxLifetime: 1h
  ConnMaxIdleTime: 30m
```

### Query Optimization

```text
// Index optimization for frequently queried fields
CREATE INDEX idx_origins_domain ON origins(domain);
CREATE INDEX idx_storages_project_priority ON storages(project_id, priority);
CREATE INDEX idx_media_requests_created ON media_requests(created_at);

// Composite indexes for complex queries
CREATE INDEX idx_origins_project_domain ON origins(project_id, domain);
```

### Database Partitioning

```sql
-- Partition large tables by date
CREATE TABLE media_logs (
    id BIGINT AUTO_INCREMENT,
    request_id VARCHAR(36),
    created_at TIMESTAMP,
    -- other fields
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (YEAR(created_at)) (
    PARTITION p2023 VALUES LESS THAN (2024),
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p2025 VALUES LESS THAN (2026)
);
```

## Storage Optimization

### Local Storage

```bash
# Use SSD for cache directory
mount /dev/nvme0n1 /var/cache/mediax

# Optimize filesystem
mount -o noatime,nodiratime /dev/sdb1 /var/media
```

### S3 Optimization

```text
// S3 configuration for performance
S3Config{
    Region:          "us-west-2",
    MaxRetries:      3,
    MaxConnections:  100,
    Timeout:         30 * time.Second,
    PartSize:        64 * 1024 * 1024, // 64MB
    Concurrency:     10,
}
```

### CDN Configuration

```text
// CloudFront distribution settings
{
    "Origins": [{
        "DomainName": "your-mediax-server.com",
        "CustomOriginConfig": {
            "HTTPPort": 8080,
            "OriginProtocolPolicy": "http-only"
        }
    }],
    "DefaultCacheBehavior": {
        "TargetOriginId": "mediax-origin",
        "ViewerProtocolPolicy": "redirect-to-https",
        "CachePolicyId": "custom-cache-policy",
        "TTL": {
            "DefaultTTL": 86400,
            "MaxTTL": 31536000
        }
    }
}
```

## Processing Optimization

### Parallel Processing

```text
// Goroutine pool for concurrent processing
type WorkerPool struct {
    workers    int
    jobQueue   chan Job
    workerPool chan chan Job
    quit       chan bool
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers:    workers,
        jobQueue:   make(chan Job, workers*2),
        workerPool: make(chan chan Job, workers),
        quit:       make(chan bool),
    }
}
```

### FFmpeg Optimization

```bash
# FFmpeg hardware acceleration
ffmpeg -hwaccel cuda -i input.mp4 -c:v h264_nvenc output.mp4

# Multi-threading
ffmpeg -threads 4 -i input.mp4 output.mp4

# Memory optimization
ffmpeg -analyzeduration 100M -probesize 100M -i input.mp4 output.mp4
```

### ImageMagick Optimization

```bash
# ImageMagick resource limits
export MAGICK_MEMORY_LIMIT=2GB
export MAGICK_MAP_LIMIT=4GB
export MAGICK_DISK_LIMIT=8GB
export MAGICK_THREAD_LIMIT=4

# Policy configuration in /etc/ImageMagick-6/policy.xml
<policy domain="resource" name="memory" value="2GiB"/>
<policy domain="resource" name="map" value="4GiB"/>
<policy domain="resource" name="disk" value="8GiB"/>
<policy domain="resource" name="thread" value="4"/>
```

## Monitoring and Metrics

### Application Metrics

```text
// Prometheus metrics
var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mediax_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "status"},
    )
    
    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mediax_processing_duration_seconds",
            Help: "Processing duration in seconds",
        },
        []string{"media_type", "operation"},
    )
)
```

### System Monitoring

```yaml
# Prometheus configuration
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'mediax'
    static_configs:
      - targets: ['mediax:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']
```

### Grafana Dashboards

```json
{
  "dashboard": {
    "title": "MediaX Performance",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(mediax_requests_total[5m])",
            "legendFormat": "{{method}} {{status}}"
          }
        ]
      },
      {
        "title": "Processing Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(mediax_processing_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

## Performance Testing

### Load Testing with Artillery

```yaml
# artillery-config.yml
config:
  target: 'http://localhost:8080'
  phases:
    - duration: 60
      arrivalRate: 10
    - duration: 120
      arrivalRate: 50
    - duration: 60
      arrivalRate: 100

scenarios:
  - name: "Image Processing"
    weight: 70
    flow:
      - get:
          url: "/images/sample.jpg?w=800&h=600&f=webp"
  
  - name: "Video Thumbnail"
    weight: 20
    flow:
      - get:
          url: "/videos/sample.mp4?f=jpg&t=30"
  
  - name: "Audio Conversion"
    weight: 10
    flow:
      - get:
          url: "/audio/sample.mp3?f=flac&q=90"
```

### Benchmark Testing

```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:8080/images/test.jpg?w=800&h=600

# wrk
wrk -t12 -c400 -d30s http://localhost:8080/images/test.jpg?w=800&h=600

# Custom Go benchmark
go test -bench=. -benchmem ./...
```

## Optimization Checklist

### Application Level

- [ ] Enable file caching
- [ ] Implement memory caching for metadata
- [ ] Use connection pooling
- [ ] Optimize database queries
- [ ] Enable compression
- [ ] Implement proper error handling
- [ ] Use goroutine pools for concurrent processing

### Infrastructure Level

- [ ] Use SSD for cache storage
- [ ] Configure CDN
- [ ] Set up load balancing
- [ ] Optimize network settings
- [ ] Configure proper resource limits
- [ ] Enable monitoring and alerting
- [ ] Implement health checks

### Database Level

- [ ] Create proper indexes
- [ ] Optimize connection settings
- [ ] Implement query caching
- [ ] Consider read replicas
- [ ] Partition large tables
- [ ] Regular maintenance tasks

### Security and Performance

- [ ] Implement rate limiting
- [ ] Use HTTPS/TLS
- [ ] Configure CORS properly
- [ ] Validate input parameters
- [ ] Implement request timeouts
- [ ] Log performance metrics

## Troubleshooting Performance Issues

### Common Performance Problems

1. **High Memory Usage**
   ```bash
   # Check memory usage
   docker stats mediax
   
   # Analyze Go heap
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

2. **Slow Database Queries**
   ```sql
   -- Enable slow query log
   SET GLOBAL slow_query_log = 'ON';
   SET GLOBAL long_query_time = 1;
   
   -- Analyze queries
   EXPLAIN SELECT * FROM origins WHERE domain = 'example.com';
   ```

3. **High CPU Usage**
   ```bash
   # Profile CPU usage
   go tool pprof http://localhost:8080/debug/pprof/profile
   
   # Check system load
   top -p $(pgrep mediax)
   ```

4. **Storage I/O Issues**
   ```bash
   # Monitor disk I/O
   iostat -x 1
   
   # Check disk space
   df -h /var/cache/mediax
   ```

### Performance Monitoring Commands

```bash
# Real-time monitoring
watch -n 1 'curl -s http://localhost:8080/metrics | grep mediax_requests_total'

# Log analysis
tail -f /var/log/mediax/access.log | grep "processing_time"

# Resource usage
docker exec mediax top -b -n 1 | head -20
```