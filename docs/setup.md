# Setup and Requirements

## System Requirements

- **Go**: Version 1.23.5 or higher
- **Database**: MySQL 5.7+ (primary), SQLite, or SQL Server
- **External Tools**:
  - **FFmpeg**: Required for video and audio processing
  - **ImageMagick**: Required for image processing
- **Operating System**: Linux, macOS, or Windows

## External Dependencies Installation

### Ubuntu/Debian
```bash
# Install FFmpeg
sudo apt update
sudo apt install ffmpeg

# Install ImageMagick
sudo apt install imagemagick

# Install MySQL
sudo apt install mysql-server
```

### macOS
```bash
# Install FFmpeg and ImageMagick using Homebrew
brew install ffmpeg imagemagick

# Install MySQL
brew install mysql
```

### Windows
1. Download and install FFmpeg from https://ffmpeg.org/download.html
2. Download and install ImageMagick from https://imagemagick.org/script/download.php#windows
3. Install MySQL from https://dev.mysql.com/downloads/mysql/

## Go Dependencies Installation

```bash
# Clone the repository
git clone <repository-url>
cd mediax

# Install Go dependencies
go mod download
```

## Database Setup

1. Create a MySQL database:
```sql
CREATE DATABASE mediax;
CREATE USER 'mediax_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON mediax.* TO 'mediax_user'@'localhost';
FLUSH PRIVILEGES;
```

2. The application will automatically create the required tables on first run.