# Docker Deployment

## Using Docker Compose

### Basic Docker Compose Setup

Create a `docker-compose.yml` file:

```yaml
version: '3.8'
services:
  mediax:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - DB_USER=mediax
      - DB_PASSWORD=password
      - DB_NAME=mediax
    depends_on:
      - mysql
    volumes:
      - ./media:/var/media
      - ./cache:/var/cache

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=rootpassword
      - MYSQL_DATABASE=mediax
      - MYSQL_USER=mediax
      - MYSQL_PASSWORD=password
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

### Production Docker Compose

For production environments:

```yaml
version: '3.8'
services:
  mediax:
    image: mediax:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mysql
      - DB_USER=mediax
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=mediax
      - REDIS_URL=redis://redis:6379
    depends_on:
      - mysql
      - redis
    volumes:
      - media_storage:/var/media
      - cache_storage:/var/cache
    networks:
      - mediax-network

  mysql:
    image: mysql:8.0
    restart: unless-stopped
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=mediax
      - MYSQL_USER=mediax
      - MYSQL_PASSWORD=${DB_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
    networks:
      - mediax-network

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis_data:/data
    networks:
      - mediax-network

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - mediax
    networks:
      - mediax-network

volumes:
  mysql_data:
  redis_data:
  media_storage:
  cache_storage:

networks:
  mediax-network:
    driver: bridge
```

## Dockerfile

### Development Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o mediax

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ffmpeg imagemagick ca-certificates

WORKDIR /app
COPY --from=builder /app/mediax .
COPY config.yml .

EXPOSE 8080
CMD ["./mediax"]
```

### Production Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mediax

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ffmpeg imagemagick ca-certificates tzdata \
    && addgroup -g 1001 -S mediax \
    && adduser -u 1001 -S mediax -G mediax

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/mediax .
COPY config.yml .

# Create necessary directories
RUN mkdir -p /var/media /var/cache \
    && chown -R mediax:mediax /app /var/media /var/cache

USER mediax

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./mediax"]
```

## Multi-stage Build with Optimization

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev upx

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
    -o mediax

# Compress binary
RUN upx --best --lzma mediax

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    imagemagick \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S mediax \
    && adduser -u 1001 -S mediax -G mediax

WORKDIR /app

# Copy binary and config
COPY --from=builder /app/mediax .
COPY config.yml .

# Create directories and set permissions
RUN mkdir -p /var/media /var/cache /var/logs \
    && chown -R mediax:mediax /app /var/media /var/cache /var/logs

USER mediax

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./mediax"]
```

## Environment Variables

### Database Configuration

```bash
# MySQL
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=mediax
DB_USER=mediax
DB_PASSWORD=your_password

# PostgreSQL
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mediax
DB_USER=mediax
DB_PASSWORD=your_password
DB_SSLMODE=disable

# SQLite
DB_TYPE=sqlite
DB_PATH=/var/data/mediax.db
```

### Application Configuration

```bash
# Server
HTTP_HOST=0.0.0.0
HTTP_PORT=8080
HTTP_READ_TIMEOUT=30s
HTTP_WRITE_TIMEOUT=30s

# Storage
STORAGE_PATH=/var/media
CACHE_PATH=/var/cache

# AWS S3
AWS_REGION=us-west-2
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your_redis_password

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Nginx Configuration

### Basic Nginx Config

```nginx
events {
    worker_connections 1024;
}

http {
    upstream mediax {
        server mediax:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;

        client_max_body_size 100M;

        location / {
            proxy_pass http://mediax;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

### Production Nginx Config with SSL

```nginx
events {
    worker_connections 1024;
}

http {
    upstream mediax {
        server mediax:8080;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

        client_max_body_size 100M;

        # Rate limiting
        limit_req zone=api burst=20 nodelay;

        # Caching for processed media
        location ~* \.(jpg|jpeg|png|gif|webp|avif|mp4|webm|mp3|flac)$ {
            proxy_pass http://mediax;
            proxy_cache_valid 200 1d;
            proxy_cache_key $uri$is_args$args;
            add_header X-Cache-Status $upstream_cache_status;
        }

        location / {
            proxy_pass http://mediax;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

## Deployment Commands

### Build and Deploy

```bash
# Build the image
docker build -t mediax:latest .

# Run with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f mediax

# Scale the service
docker-compose up -d --scale mediax=3

# Update the service
docker-compose pull
docker-compose up -d
```

### Production Deployment

```bash
# Build for production
docker build -f Dockerfile.prod -t mediax:prod .

# Tag for registry
docker tag mediax:prod your-registry.com/mediax:latest

# Push to registry
docker push your-registry.com/mediax:latest

# Deploy to production
docker-compose -f docker-compose.prod.yml up -d
```

## Monitoring and Logging

### Docker Compose with Monitoring

```yaml
version: '3.8'
services:
  mediax:
    image: mediax:latest
    # ... other config

  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana

volumes:
  grafana_data:
```

### Log Configuration

```yaml
# In docker-compose.yml
services:
  mediax:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Troubleshooting

### Common Issues

1. **Permission Issues**
   ```bash
   # Fix file permissions
   docker-compose exec mediax chown -R mediax:mediax /var/media
   ```

2. **Database Connection**
   ```bash
   # Check database connectivity
   docker-compose exec mediax ping mysql
   ```

3. **Memory Issues**
   ```bash
   # Increase memory limits
   docker-compose up -d --memory=2g
   ```

4. **Storage Issues**
   ```bash
   # Check disk space
   docker system df
   docker system prune
   ```

### Health Checks

```bash
# Check container health
docker-compose ps

# Check application health
curl http://localhost:8080/health

# Check logs
docker-compose logs --tail=100 mediax
```