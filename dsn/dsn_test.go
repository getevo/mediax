package dsn

import (
	"reflect"
	"testing"
)

// e/ef/Lunar_Landing_Training_vehicle_piloted_by_Neil_Armstrong_during_training_%287944972374%29.jpg
// Example: https://upload.wikimedia.org/wikipedia/commons
// Example: http://127.0.0.1:8080/wikipedia/commons
// Example: https://upload.wikimedia.org/wikipedia/commons?header[Authorization]=Bearer TOKEN&Query[download]=true
type HTTP struct {
	DSN    string `dsn:"http(s)://$Path"`
	Scheme string
	Path   string
	Debug  bool `default:"false"`
	Params map[string]string
}

// Example: fs:///mnt/storage/data?Debug=true
// Example: fs:///mnt/storage/data?type=SSD
type FileSystem struct {
	DSN    string `dsn:"fs://$Path"`
	Scheme string
	Path   string
	Debug  bool `default:"false"`
	Params map[string]string
}

// Example: s3://access_key:secret@https://s3.us-west-2.amazonaws.com/mybucket?Region=us-west-2=&Debug=true&IgnoreSSL=true
// Example: s3://access_key:secret@https://s3.us-west-2.amazonaws.com/mybucket?Region=us-west-2&Debug=true&IgnoreSSL=true&BasePath=my/prefix
type S3 struct {
	DSN       string `dsn:"s3://$AccessKey:$SecretKey@$Endpoint/$Bucket"`
	Scheme    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	BasePath  string `default:""`
	Debug     bool   `default:"false"`
	IgnoreSSL bool   `default:"false"`
	Params    map[string]string
}

// Example: sftp://user:password@example.com:23/home/usr?Debug=true
// Example: sftp://user:password@example.com:23/home/usr?Debug=true&Passive=true
type SFTP struct {
	DSN      string `dsn:"sftp://$Username:$Password@$Host:$Port/$BasePath"`
	Scheme   string
	Username string
	Password string
	Host     string
	Port     int    `default:"23"`
	BasePath string `default:""`
	Debug    bool   `default:"false"`
	Params   map[string]string
}

// Example: ftp://user:password@example.com:22/home/usr?Debug=true
// Example: ftp://user:password@example.com:22/home/usr?Debug=true&Passive=true
type FTP struct {
	DSN      string `dsn:"ftp://$Username:$Password@$Host:$Port/$BasePath"`
	Scheme   string
	Username string
	Password string
	Host     string
	Port     int    `default:"22"`
	BasePath string `default:""`
	Debug    bool   `default:"false"`
	Params   map[string]string
}

func TestParseFileSystem(t *testing.T) {
	tests := []struct {
		input string
		want  FileSystem
	}{
		{
			"fs:///mnt/storage/data?Debug=true",
			FileSystem{
				Scheme: "fs",
				Path:   "/mnt/storage/data",
				Debug:  true,
				Params: map[string]string{"Debug": "true"},
			},
		},
		{
			"fs:///mnt/storage/data?type=SSD",
			FileSystem{
				Scheme: "fs",
				Path:   "/mnt/storage/data",
				Debug:  false,
				Params: map[string]string{"type": "SSD"},
			},
		},
	}

	for _, tt := range tests {
		var cfg FileSystem
		err := ParseDSN(tt.input, &cfg)
		if err != nil {
			t.Errorf("ParseDSN(%q) failed: %v", tt.input, err)
			continue
		}
		cfg.DSN = "" // ignore DSN in comparison
		if !reflect.DeepEqual(cfg, tt.want) {
			t.Errorf("ParseDSN(%q) = %+v, want %+v", tt.input, cfg, tt.want)
		}
	}
}

func TestParseHTTP(t *testing.T) {
	tests := []struct {
		input string
		want  HTTP
	}{
		{
			input: "https://upload.wikimedia.org/wikipedia/commons",
			want: HTTP{
				Scheme: "https",
				Path:   "upload.wikimedia.org/wikipedia/commons",
				Debug:  false,
				Params: map[string]string{},
			},
		},
		{
			input: "http://127.0.0.1:8080/wikipedia/commons",
			want: HTTP{
				Scheme: "http",
				Path:   "127.0.0.1:8080/wikipedia/commons",
				Debug:  false,
				Params: map[string]string{},
			},
		},
		{
			input: "https://upload.wikimedia.org/wikipedia/commons?header[Authorization]=Bearer TOKEN&Query[download]=true",
			want: HTTP{
				Scheme: "https",
				Path:   "upload.wikimedia.org/wikipedia/commons",
				Debug:  false,
				Params: map[string]string{
					"header[Authorization]": "Bearer TOKEN",
					"Query[download]":       "true",
				},
			},
		},
	}

	for _, tt := range tests {
		var cfg HTTP
		err := ParseDSN(tt.input, &cfg)
		if err != nil {
			t.Errorf("ParseDSN(%q) failed: %v", tt.input, err)
			continue
		}
		cfg.DSN = "" // omit for comparison
		if !reflect.DeepEqual(cfg, tt.want) {
			t.Errorf("ParseDSN(%q) = %+v\nwant %+v", tt.input, cfg, tt.want)
		}
	}
}

func TestParseS3(t *testing.T) {
	tests := []struct {
		input string
		want  S3
	}{
		{
			"s3://access_key:secret@https://s3.us-west-2.amazonaws.com/mybucket?Region=us-west-2&Debug=true&IgnoreSSL=true",
			S3{
				Scheme:    "s3",
				AccessKey: "access_key",
				SecretKey: "secret",
				Region:    "us-west-2",
				Bucket:    "mybucket",
				Endpoint:  "https://s3.us-west-2.amazonaws.com",
				Debug:     true,
				IgnoreSSL: true,
				BasePath:  "",
				Params: map[string]string{
					"Region":    "us-west-2",
					"Debug":     "true",
					"IgnoreSSL": "true",
				},
			},
		},
		{
			"s3://access_key:secret@https://s3.us-west-2.amazonaws.com/mybucket?Region=us-west-2&Debug=true&IgnoreSSL=true&BasePath=my/prefix",
			S3{
				Scheme:    "s3",
				AccessKey: "access_key",
				SecretKey: "secret",
				Region:    "us-west-2",
				Bucket:    "mybucket",
				BasePath:  "my/prefix",
				Endpoint:  "https://s3.us-west-2.amazonaws.com",
				Debug:     true,
				IgnoreSSL: true,
				Params: map[string]string{
					"Region":    "us-west-2",
					"Debug":     "true",
					"IgnoreSSL": "true",
					"BasePath":  "my/prefix",
				},
			},
		},
	}

	for _, tt := range tests {
		var cfg S3
		err := ParseDSN(tt.input, &cfg)
		if err != nil {
			t.Errorf("ParseDSN(%q) failed: %v", tt.input, err)
			continue
		}
		cfg.DSN = "" // ignore DSN in comparison
		if !reflect.DeepEqual(cfg, tt.want) {
			t.Errorf("ParseDSN(%q) = %+v, want %+v", tt.input, cfg, tt.want)
		}
	}
}

func TestParseSFTP(t *testing.T) {
	tests := []struct {
		input string
		want  SFTP
	}{
		{
			"sftp://user:password@example.com:23/home/usr?Debug=true",
			SFTP{
				Scheme:   "sftp",
				Username: "user",
				Password: "password",
				Host:     "example.com",
				Port:     23,
				BasePath: "home/usr",
				Debug:    true,
				Params:   map[string]string{"Debug": "true"},
			},
		},
		{
			"sftp://user:password@example.com:23/home/usr?Debug=true&Passive=true",
			SFTP{
				Scheme:   "sftp",
				Username: "user",
				Password: "password",
				Host:     "example.com",
				Port:     23,
				BasePath: "home/usr",
				Debug:    true,
				Params:   map[string]string{"Debug": "true", "Passive": "true"},
			},
		},
	}

	for _, tt := range tests {
		var cfg SFTP
		err := ParseDSN(tt.input, &cfg)
		if err != nil {
			t.Errorf("ParseDSN(%q) failed: %v", tt.input, err)
			continue
		}
		cfg.DSN = "" // ignore DSN in comparison
		if !reflect.DeepEqual(cfg, tt.want) {
			t.Errorf("ParseDSN(%q) = %+v, want %+v", tt.input, cfg, tt.want)
		}
	}
}

func TestParseFTP(t *testing.T) {
	tests := []struct {
		input string
		want  FTP
	}{
		{
			"ftp://user:password@example.com:22/home/usr?Debug=true",
			FTP{
				Scheme:   "ftp",
				Username: "user",
				Password: "password",
				Host:     "example.com",
				Port:     22,
				BasePath: "home/usr",
				Debug:    true,
				Params:   map[string]string{"Debug": "true"},
			},
		},
		{
			"ftp://user:password@example.com:22/home/usr?Debug=true&Passive=true",
			FTP{
				Scheme:   "ftp",
				Username: "user",
				Password: "password",
				Host:     "example.com",
				Port:     22,
				BasePath: "home/usr",
				Debug:    true,
				Params:   map[string]string{"Debug": "true", "Passive": "true"},
			},
		},
	}

	for _, tt := range tests {
		var cfg FTP
		err := ParseDSN(tt.input, &cfg)
		if err != nil {
			t.Errorf("ParseDSN(%q) failed: %v", tt.input, err)
			continue
		}
		cfg.DSN = "" // ignore DSN in comparison
		if !reflect.DeepEqual(cfg, tt.want) {
			t.Errorf("ParseDSN(%q) = %+v, want %+v", tt.input, cfg, tt.want)
		}
	}
}
