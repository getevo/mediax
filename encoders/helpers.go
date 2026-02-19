package encoders

import "time"

const (
	// Video preview constants
	chunkDuration       = 4.0  // seconds per preview chunk
	maxPreviewDuration  = 20.0 // maximum preview duration in seconds
	ffmpegCRF           = "28" // FFmpeg CRF for preview compression (higher = smaller file)
	maxConcurrentChunks = 4    // maximum concurrent FFmpeg chunk extraction goroutines

	// Command timeout constants (#5)
	imageConvertTimeout  = 60 * time.Second  // timeout for ImageMagick convert/identify
	officeConvertTimeout = 120 * time.Second // timeout for LibreOffice/pdftoppm conversions
)

// truncateOutput caps command stderr/stdout at 500 characters to prevent log bloat (#6).
func truncateOutput(output []byte) string {
	const maxLen = 500
	s := string(output)
	if len(s) > maxLen {
		return s[:maxLen] + "... (truncated)"
	}
	return s
}
