# Configuration and Running

## Configuration

### Basic Configuration

MediaX uses a `config.yml` file for basic server and database configuration:

```yaml
Database:
  Type: mysql
  Server: localhost:3306
  Database: "mediax"
  Username: mediax_user
  Password: "your_password"
  Params: "parseTime=true"
  MaxOpenConns: 100
  MaxIdleConns: 10
  ConnMaxLifTime: 1h

HTTP:
  Host: 0.0.0.0
  Port: 8080
  BodyLimit: 25mb
  ReadTimeout: 1s
  WriteTimeout: 5s
```

### Database Configuration

MediaX uses database-driven configuration for domains, storage backends, and processing profiles. The main configuration tables are:

- **Projects**: Group related domains and storage configurations
- **Origins**: Define allowed domains and their settings
- **Storages**: Configure storage backends (local, S3, HTTP)
- **VideoProfiles**: Define video encoding profiles

## How to Run

### Development Mode

```bash
# Run the application
go run main.go

# Or build and run
go build -o mediax
./mediax
```

### Production Mode

```bash
# Build for production
go build -ldflags="-s -w" -o mediax

# Run with production config
./mediax
```

The server will start on `http://localhost:8080` by default.

## Building

```bash
# Build for current platform
go build -o mediax

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o mediax-linux

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o mediax.exe
```