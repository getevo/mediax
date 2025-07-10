package encoders

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/getevo/evo/v2/lib/log"
	"mediax/apps/media"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// getVideoDuration gets the duration of a video file in seconds using ffprobe
func getVideoDuration(filePath string) (float64, error) {
	// Set timeout for ffprobe command (10 seconds should be enough)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "csv=p=0", filePath)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("ffprobe timed out after 10 seconds while getting video duration")
		}
		return 0, fmt.Errorf("failed to get video duration: %v", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}

// getQualityDimensions returns width and height for quality presets
func getQualityDimensions(quality string) (int, int) {
	switch strings.ToLower(quality) {
	case "480p":
		return 854, 480
	case "720p":
		return 1280, 720
	case "1080p":
		return 1920, 1080
	case "4k":
		return 3840, 2160
	default:
		return 854, 480 // default to 480p
	}
}

// generateCacheKey generates a unique cache key for the processed file
func generateCacheKey(originalPath string, options *media.Options) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%s_%s_%s_%d_%s", originalPath, options.Preview, options.Thumbnail, options.SS, options.OutputFormat)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// generatePreview creates a preview clip by intelligently splitting video into chunks
func generatePreview(input *media.Request) error {
	if input.Options.Preview == "" {
		return nil
	}

	// Generate cache key and check if preview already exists
	cacheKey := generateCacheKey(input.OriginalFilePath, input.Options)
	quality := input.Options.Preview
	switch quality {
	case "480p", "720p", "1080p", "4k":
		break
	default:
		quality = "480p"
	}

	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "previews")
	os.MkdirAll(cacheDir, 0755)

	previewPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.mp4", cacheKey, quality))

	// Check if cached version exists
	if _, err := os.Stat(previewPath); err == nil {
		if input.Debug {
			log.Debug("Cache hit for video preview", "trace_id", input.TraceID, "cache_key", cacheKey, "quality", quality, "preview_path", previewPath)
			input.Request.Set("X-Debug-Cache-Status", "HIT")
			input.Request.Set("X-Debug-Cache-Key", cacheKey)
			input.Request.Set("X-Debug-Cache-Path", previewPath)
		}
		input.ProcessedFilePath = previewPath
		return nil
	}

	if input.Debug {
		log.Debug("Cache miss for video preview", "trace_id", input.TraceID, "cache_key", cacheKey, "quality", quality, "preview_path", previewPath)
		input.Request.Set("X-Debug-Cache-Status", "MISS")
		input.Request.Set("X-Debug-Cache-Key", cacheKey)
		input.Request.Set("X-Debug-Cache-Path", previewPath)
	}

	// Get video duration
	duration, err := getVideoDuration(input.StagedFilePath)
	if err != nil {
		return fmt.Errorf("failed to get video duration: %v", err)
	}

	// Calculate chunk parameters
	chunkDuration := 4.0                                 // 4 seconds per chunk
	maxPreviewDuration := 20.0                           // maximum 20 seconds
	maxChunks := int(maxPreviewDuration / chunkDuration) // 5 chunks max

	// Calculate how many chunks we can extract
	totalPossibleChunks := int(duration / chunkDuration)
	if totalPossibleChunks < 1 {
		totalPossibleChunks = 1
	}

	chunksToExtract := maxChunks
	if totalPossibleChunks < maxChunks {
		chunksToExtract = totalPossibleChunks
	}

	// Calculate interval between chunks for intelligent distribution
	interval := duration / float64(chunksToExtract)

	width, height := getQualityDimensions(quality)

	// Create temporary directory for chunks
	tempDir := filepath.Join(cacheDir, "temp_"+cacheKey)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Extract chunks in parallel
	var wg sync.WaitGroup
	chunkPaths := make([]string, chunksToExtract)
	errors := make([]error, chunksToExtract)

	for i := 0; i < chunksToExtract; i++ {
		wg.Add(1)
		go func(chunkIndex int) {
			defer wg.Done()

			startTime := float64(chunkIndex) * interval
			chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d.mp4", chunkIndex))

			// Set timeout for chunk extraction (60 seconds should be enough)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Extract chunk with no audio, compression, and quality scaling
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-ss", fmt.Sprintf("%.2f", startTime),
				"-i", input.StagedFilePath,
				"-t", fmt.Sprintf("%.2f", chunkDuration),
				"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", width, height, width, height),
				"-c:v", "libx264",
				"-preset", "fast",
				"-crf", "28", // Higher CRF for more compression
				"-an", // Remove audio
				"-y", chunkPath)

			if err := cmd.Run(); err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					errors[chunkIndex] = fmt.Errorf("chunk %d extraction timed out after 60 seconds", chunkIndex)
				} else {
					errors[chunkIndex] = fmt.Errorf("failed to extract chunk %d: %v", chunkIndex, err)
				}
				return
			}

			chunkPaths[chunkIndex] = chunkPath
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return fmt.Errorf("chunk %d extraction failed: %v", i, err)
		}
	}

	// Create concat file
	concatFile := filepath.Join(tempDir, "concat.txt")
	concatContent := ""
	for _, chunkPath := range chunkPaths {
		if chunkPath != "" {
			concatContent += fmt.Sprintf("file '%s'\n", chunkPath)
		}
	}

	err = os.WriteFile(concatFile, []byte(concatContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %v", err)
	}

	// Concatenate chunks with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
		"-c", "copy",
		"-y", previewPath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("chunk concatenation timed out after 30 seconds")
		}
		return fmt.Errorf("failed to concatenate chunks: %v", err)
	}

	input.ProcessedFilePath = previewPath
	return nil
}

// parseThumbnailDimensions parses thumbnail parameter to get width and height
func parseThumbnailDimensions(thumbnail string) (int, int, bool) {
	// Check if it's custom dimensions (e.g., "640x480")
	if strings.Contains(thumbnail, "x") {
		parts := strings.Split(thumbnail, "x")
		if len(parts) == 2 {
			width, err1 := strconv.Atoi(parts[0])
			height, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && width > 0 && height > 0 {
				return width, height, true // true means custom dimensions
			}
		}
	}

	// Use quality presets
	width, height := getQualityDimensions(thumbnail)
	return width, height, false // false means quality preset
}

// getImageFormat returns the appropriate format and extension
func getImageFormat(outputFormat string) (string, string) {
	switch strings.ToLower(outputFormat) {
	case "webp":
		return "webp", "webp"
	case "jpg", "jpeg":
		return "jpg", "jpg"
	case "png":
		return "png", "png"
	case "avif":
		return "avif", "avif"
	default:
		return "jpg", "jpg" // default to jpeg
	}
}

// getImageMimeType returns the appropriate MIME type for image formats
func getImageMimeType(outputFormat string) string {
	switch strings.ToLower(outputFormat) {
	case "webp":
		return "image/webp"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "avif":
		return "image/avif"
	default:
		return "image/jpeg" // default to jpeg
	}
}

// generateThumbnail creates a thumbnail from the video
func generateThumbnail(input *media.Request) error {
	if input.Options.Thumbnail == "" {
		return nil
	}

	// Determine output format (default to jpeg)
	outputFormat := input.Options.OutputFormat
	if outputFormat == "" {
		outputFormat = "jpeg"
	}

	// Generate cache key and check if thumbnail already exists
	cacheKey := generateCacheKey(input.OriginalFilePath, input.Options)
	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "thumbnails")
	os.MkdirAll(cacheDir, 0755)
	// Determine final file extension
	_, finalExtension := getImageFormat(outputFormat)
	finalPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.%s", cacheKey, input.Options.Thumbnail, finalExtension))

	// Check if cached version exists
	if _, err := os.Stat(finalPath); err == nil {
		if input.Debug {
			log.Debug("Cache hit for video thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
			input.Request.Set("X-Debug-Thumbnail-Cache-Status", "HIT")
			input.Request.Set("X-Debug-Thumbnail-Cache-Key", cacheKey)
			input.Request.Set("X-Debug-Thumbnail-Cache-Path", finalPath)
		}
		input.ProcessedFilePath = finalPath
		input.ProcessedMimeType = getImageMimeType(outputFormat)
		return nil
	}

	if input.Debug {
		log.Debug("Cache miss for video thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
		input.Request.Set("X-Debug-Thumbnail-Cache-Status", "MISS")
		input.Request.Set("X-Debug-Thumbnail-Cache-Key", cacheKey)
		input.Request.Set("X-Debug-Thumbnail-Cache-Path", finalPath)
	}

	// Determine timestamp (use ss if provided, otherwise middle of video)
	timestamp := float64(input.Options.SS)
	if input.Options.SS == 0 {
		duration, err := getVideoDuration(input.StagedFilePath)
		if err != nil {
			return fmt.Errorf("failed to get video duration: %v", err)
		}
		timestamp = duration / 2
	}

	// Step 1: Generate JPEG thumbnail with maximum scale using FFmpeg
	jpegPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_temp.jpg", cacheKey, input.Options.Thumbnail))

	// Set timeout for FFmpeg command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate high-quality JPEG with maximum scale (no specific dimensions)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", input.StagedFilePath,
		"-vframes", "1",
		"-q:v", "2", // High quality JPEG
		"-y", jpegPath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("thumbnail generation timed out after 30 seconds")
		}
		return fmt.Errorf("failed to extract thumbnail: %v", err)
	}

	// Step 2: Use ImageMagick convert to change format and size based on user input
	args := []string{jpegPath}

	// Parse thumbnail parameter for size
	if strings.Contains(input.Options.Thumbnail, "x") {
		// Custom dimensions (e.g., "60x60")
		args = append(args, "-resize", input.Options.Thumbnail+"^")
		args = append(args, "-gravity", "center")
		args = append(args, "-crop", input.Options.Thumbnail+"+0+0")
	} else {
		// Quality presets (e.g., "1080p")
		width, height := getQualityDimensions(input.Options.Thumbnail)
		args = append(args, "-resize", fmt.Sprintf("%dx%d", width, height))
	}

	// Apply quality if specified
	if input.Options.Quality > 0 {
		args = append(args, "-quality", fmt.Sprintf("%d", input.Options.Quality))
	}

	// Set output file
	args = append(args, finalPath)

	// Execute ImageMagick convert
	convertCmd := exec.Command("convert", args...)
	output, err := convertCmd.CombinedOutput()
	if err != nil {
		// Clean up temporary JPEG file
		os.Remove(jpegPath)
		return fmt.Errorf("ImageMagick convert error: %v\noutput: %s", err, output)
	}

	// Clean up temporary JPEG file
	os.Remove(jpegPath)

	input.ProcessedFilePath = finalPath
	input.ProcessedMimeType = getImageMimeType(outputFormat)
	return nil
}

// processVideo handles both preview and thumbnail generation
func processVideo(input *media.Request) error {
	if input.Debug {
		log.Debug("Starting video processing", "trace_id", input.TraceID, "preview", input.Options.Preview, "thumbnail", input.Options.Thumbnail)
	}

	// Handle preview generation
	if input.Options.Preview != "" {
		if input.Debug {
			log.Debug("Processing video preview", "trace_id", input.TraceID, "quality", input.Options.Preview)
		}
		return generatePreview(input)
	}

	// Handle thumbnail generation
	if input.Options.Thumbnail != "" {
		if input.Debug {
			log.Debug("Processing video thumbnail", "trace_id", input.TraceID, "thumbnail", input.Options.Thumbnail)
		}
		return generateThumbnail(input)
	}

	// No processing needed
	if input.Debug {
		log.Debug("No video processing needed", "trace_id", input.TraceID)
	}
	return nil
}

// Video encoders with preview and thumbnail support
var Mp4 = media.Encoder{
	Mime:      "video/mp4",
	Processor: processVideo,
}

var Webm = media.Encoder{
	Mime:      "video/webm",
	Processor: processVideo,
}

var Avi = media.Encoder{
	Mime:      "video/x-msvideo",
	Processor: processVideo,
}

var Mov = media.Encoder{
	Mime:      "video/quicktime",
	Processor: processVideo,
}

var Mkv = media.Encoder{
	Mime:      "video/x-matroska",
	Processor: processVideo,
}

var Flv = media.Encoder{
	Mime:      "video/x-flv",
	Processor: processVideo,
}

var Wmv = media.Encoder{
	Mime:      "video/x-ms-wmv",
	Processor: processVideo,
}

var M4v = media.Encoder{
	Mime:      "video/x-m4v",
	Processor: processVideo,
}

var ThreeGp = media.Encoder{
	Mime:      "video/3gpp",
	Processor: processVideo,
}

var Ogv = media.Encoder{
	Mime:      "video/ogg",
	Processor: processVideo,
}
