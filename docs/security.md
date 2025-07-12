# Security Considerations

## Authentication and Authorization

### API Authentication

Implement proper authentication for admin API endpoints:

```text
// JWT token validation middleware
func AuthMiddleware(c *fiber.Ctx) error {
    token := c.Get("Authorization")
    if token == "" {
        return c.Status(401).JSON(fiber.Map{
            "error": "Missing authorization header",
        })
    }
    
    // Validate JWT token
    claims, err := validateJWT(token)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{
            "error": "Invalid token",
        })
    }
    
    c.Locals("user", claims)
    return c.Next()
}
```

### Role-Based Access Control

```text
// RBAC implementation
type Role string

const (
    RoleAdmin     Role = "admin"
    RoleEditor    Role = "editor"
    RoleViewer    Role = "viewer"
)

type User struct {
    ID    uint   `json:"id"`
    Email string `json:"email"`
    Role  Role   `json:"role"`
}

func RequireRole(role Role) fiber.Handler {
    return func(c *fiber.Ctx) error {
        user := c.Locals("user").(*User)
        if user.Role != role && user.Role != RoleAdmin {
            return c.Status(403).JSON(fiber.Map{
                "error": "Insufficient permissions",
            })
        }
        return c.Next()
    }
}
```

## Input Validation and Sanitization

### Parameter Validation

```text
// Input validation for media processing parameters
type MediaParams struct {
    Width   int    `validate:"min=1,max=4096"`
    Height  int    `validate:"min=1,max=4096"`
    Quality int    `validate:"min=1,max=100"`
    Format  string `validate:"oneof=jpg png gif webp avif mp4 webm"`
}

func ValidateParams(params *MediaParams) error {
    validate := validator.New()
    return validate.Struct(params)
}
```

### File Path Sanitization

```text
// Prevent path traversal attacks
func SanitizePath(path string) (string, error) {
    // Remove any path traversal attempts
    cleaned := filepath.Clean(path)
    
    // Ensure path doesn't go outside allowed directories
    if strings.Contains(cleaned, "..") {
        return "", errors.New("invalid path")
    }
    
    // Remove leading slashes
    cleaned = strings.TrimPrefix(cleaned, "/")
    
    return cleaned, nil
}
```

### Content Type Validation

```text
// Validate file content type
func ValidateContentType(file []byte, allowedTypes []string) error {
    contentType := http.DetectContentType(file)
    
    for _, allowed := range allowedTypes {
        if strings.HasPrefix(contentType, allowed) {
            return nil
        }
    }
    
    return fmt.Errorf("invalid content type: %s", contentType)
}
```

## Rate Limiting

### Application-Level Rate Limiting

```text
// Rate limiter implementation
type RateLimiter struct {
    requests map[string][]time.Time
    mutex    sync.RWMutex
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(clientIP string) bool {
    rl.mutex.Lock()
    defer rl.mutex.Unlock()
    
    now := time.Now()
    cutoff := now.Add(-rl.window)
    
    // Clean old requests
    requests := rl.requests[clientIP]
    validRequests := []time.Time{}
    for _, req := range requests {
        if req.After(cutoff) {
            validRequests = append(validRequests, req)
        }
    }
    
    // Check if limit exceeded
    if len(validRequests) >= rl.limit {
        return false
    }
    
    // Add current request
    validRequests = append(validRequests, now)
    rl.requests[clientIP] = validRequests
    
    return true
}
```

### Nginx Rate Limiting

```nginx
# Rate limiting configuration
http {
    # Define rate limit zones
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=media:10m rate=50r/s;
    
    server {
        # API endpoints - stricter limits
        location /admin/ {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://mediax;
        }
        
        # Media processing - higher limits
        location / {
            limit_req zone=media burst=100 nodelay;
            proxy_pass http://mediax;
        }
    }
}
```

## HTTPS and TLS Configuration

### TLS Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    # SSL certificates
    ssl_certificate /etc/ssl/certs/your-domain.crt;
    ssl_certificate_key /etc/ssl/private/your-domain.key;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # SSL session cache
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Referrer-Policy "strict-origin-when-cross-origin";
}
```

### Certificate Management

```bash
# Using Let's Encrypt with Certbot
certbot --nginx -d your-domain.com

# Auto-renewal
echo "0 12 * * * /usr/bin/certbot renew --quiet" | crontab -
```

## CORS Configuration

### Secure CORS Setup

```text
// CORS middleware configuration
func CORSMiddleware() fiber.Handler {
    return cors.New(cors.Config{
        AllowOrigins:     "https://your-frontend.com,https://admin.your-domain.com",
        AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
        AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Debug",
        ExposeHeaders:    "X-Trace-ID,X-Cache-Status",
        AllowCredentials: true,
        MaxAge:           86400, // 24 hours
    })
}
```

## Data Protection

### Encryption at Rest

```yaml
# Database encryption
Database:
  Type: mysql
  Server: localhost:3306
  Database: "mediax"
  Username: mediax_user
  Password: "encrypted_password"
  TLS: true
  SSLMode: "require"
  
# Storage encryption
Storage:
  Type: s3
  Bucket: "encrypted-media-bucket"
  Region: "us-west-2"
  Encryption: "AES256"
  KMSKeyID: "arn:aws:kms:us-west-2:123456789012:key/12345678-1234-1234-1234-123456789012"
```

### Encryption in Transit

```text
// TLS configuration for database connections
func configureTLS() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }
}
```

## Secrets Management

### Environment Variables

```bash
# Use environment variables for sensitive data
export DB_PASSWORD="$(cat /run/secrets/db_password)"
export JWT_SECRET="$(cat /run/secrets/jwt_secret)"
export AWS_SECRET_ACCESS_KEY="$(cat /run/secrets/aws_secret)"
```

### Docker Secrets

```yaml
# docker-compose.yml with secrets
version: '3.8'
services:
  mediax:
    image: mediax:latest
    secrets:
      - db_password
      - jwt_secret
      - aws_secret
    environment:
      - DB_PASSWORD_FILE=/run/secrets/db_password
      - JWT_SECRET_FILE=/run/secrets/jwt_secret
      - AWS_SECRET_ACCESS_KEY_FILE=/run/secrets/aws_secret

secrets:
  db_password:
    file: ./secrets/db_password.txt
  jwt_secret:
    file: ./secrets/jwt_secret.txt
  aws_secret:
    file: ./secrets/aws_secret.txt
```

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mediax-secrets
type: Opaque
data:
  db-password: <base64-encoded-password>
  jwt-secret: <base64-encoded-secret>
  aws-secret: <base64-encoded-secret>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mediax
spec:
  template:
    spec:
      containers:
      - name: mediax
        image: mediax:latest
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mediax-secrets
              key: db-password
```

## Security Headers

### Application Security Headers

```text
// Security headers middleware
func SecurityHeaders() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Prevent clickjacking
        c.Set("X-Frame-Options", "DENY")
        
        // Prevent MIME type sniffing
        c.Set("X-Content-Type-Options", "nosniff")
        
        // XSS protection
        c.Set("X-XSS-Protection", "1; mode=block")
        
        // Referrer policy
        c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        // Content Security Policy
        c.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; script-src 'self'")
        
        return c.Next()
    }
}
```

## Logging and Monitoring

### Security Event Logging

```text
// Security event logging
type SecurityEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    EventType   string    `json:"event_type"`
    ClientIP    string    `json:"client_ip"`
    UserAgent   string    `json:"user_agent"`
    RequestPath string    `json:"request_path"`
    Severity    string    `json:"severity"`
    Message     string    `json:"message"`
}

func LogSecurityEvent(eventType, clientIP, userAgent, path, severity, message string) {
    event := SecurityEvent{
        Timestamp:   time.Now(),
        EventType:   eventType,
        ClientIP:    clientIP,
        UserAgent:   userAgent,
        RequestPath: path,
        Severity:    severity,
        Message:     message,
    }
    
    log.Warn("Security event", "event", event)
}
```

### Intrusion Detection

```text
// Simple intrusion detection
type IntrusionDetector struct {
    failedAttempts map[string]int
    mutex          sync.RWMutex
    threshold      int
    window         time.Duration
}

func (id *IntrusionDetector) RecordFailedAttempt(clientIP string) bool {
    id.mutex.Lock()
    defer id.mutex.Unlock()
    
    id.failedAttempts[clientIP]++
    
    if id.failedAttempts[clientIP] >= id.threshold {
        LogSecurityEvent("intrusion_detected", clientIP, "", "", "high", 
            fmt.Sprintf("Too many failed attempts: %d", id.failedAttempts[clientIP]))
        return true
    }
    
    return false
}
```

## Vulnerability Management

### Dependency Scanning

```bash
# Go vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Docker image scanning
docker scan mediax:latest

# Trivy scanning
trivy image mediax:latest
```

### Security Updates

```dockerfile
# Regular security updates in Dockerfile
FROM alpine:latest
RUN apk update && apk upgrade && \
    apk add --no-cache ffmpeg imagemagick ca-certificates && \
    rm -rf /var/cache/apk/*
```

## Backup and Recovery

### Secure Backups

```bash
# Encrypted database backup
mysqldump --single-transaction --routines --triggers mediax | \
gpg --cipher-algo AES256 --compress-algo 1 --symmetric --output backup.sql.gpg

# S3 backup with encryption
aws s3 cp backup.sql.gpg s3://backup-bucket/mediax/ \
    --server-side-encryption AES256
```

### Disaster Recovery

```yaml
# Backup strategy configuration
Backup:
  Schedule: "0 2 * * *"  # Daily at 2 AM
  Retention: "30d"       # Keep for 30 days
  Encryption: true
  Destinations:
    - type: s3
      bucket: "mediax-backups"
      region: "us-west-2"
    - type: local
      path: "/var/backups/mediax"
```

## Security Checklist

### Application Security

- [ ] Implement proper authentication and authorization
- [ ] Validate and sanitize all inputs
- [ ] Use parameterized queries to prevent SQL injection
- [ ] Implement rate limiting
- [ ] Add security headers
- [ ] Configure CORS properly
- [ ] Use HTTPS everywhere
- [ ] Implement proper error handling (don't expose sensitive info)

### Infrastructure Security

- [ ] Use strong TLS configuration
- [ ] Implement network segmentation
- [ ] Configure firewalls properly
- [ ] Use secrets management
- [ ] Enable security logging and monitoring
- [ ] Regular security updates
- [ ] Backup and disaster recovery plan
- [ ] Vulnerability scanning

### Operational Security

- [ ] Regular security audits
- [ ] Penetration testing
- [ ] Security training for team
- [ ] Incident response plan
- [ ] Access control and privilege management
- [ ] Regular password rotation
- [ ] Multi-factor authentication
- [ ] Security documentation

## Compliance Considerations

### GDPR Compliance

```text
// Data retention policy
type DataRetentionPolicy struct {
    LogRetention   time.Duration // 90 days
    CacheRetention time.Duration // 24 hours
    BackupRetention time.Duration // 7 years
}

// Data anonymization
func AnonymizeUserData(userID string) error {
    // Remove or hash personal identifiers
    return db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
        "email": fmt.Sprintf("deleted_%s@example.com", generateHash(userID)),
        "name":  "Deleted User",
    })
}
```

### SOC 2 Compliance

- Implement access controls and monitoring
- Maintain audit logs
- Regular security assessments
- Data encryption requirements
- Incident response procedures

## Security Incident Response

### Incident Response Plan

1. **Detection**: Monitor for security events
2. **Analysis**: Assess the severity and impact
3. **Containment**: Isolate affected systems
4. **Eradication**: Remove the threat
5. **Recovery**: Restore normal operations
6. **Lessons Learned**: Document and improve

### Emergency Procedures

```bash
# Emergency shutdown
docker-compose down

# Isolate compromised container
docker network disconnect mediax-network mediax_container

# Check for indicators of compromise
grep -i "suspicious" /var/log/mediax/*.log
```