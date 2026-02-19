package encoders

import (
	"context"
	"crypto/md5"
	"encoding/json"
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

// VideoMetadata represents all video metadata information
type VideoMetadata struct {
	// Basic metadata
	Format   string  `json:"format,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Size     int64   `json:"size,omitempty"`
	Bitrate  int     `json:"bitrate,omitempty"`

	// Video stream metadata
	VideoCodec  string  `json:"video_codec,omitempty"`
	Width       int     `json:"width,omitempty"`
	Height      int     `json:"height,omitempty"`
	AspectRatio string  `json:"aspect_ratio,omitempty"`
	FrameRate   float64 `json:"frame_rate,omitempty"`
	ColorSpace  string  `json:"color_space,omitempty"`
	PixelFormat string  `json:"pixel_format,omitempty"`

	// Audio stream metadata
	AudioCodec    string `json:"audio_codec,omitempty"`
	AudioChannels int    `json:"audio_channels,omitempty"`
	SampleRate    int    `json:"sample_rate,omitempty"`

	// Subtitle information
	SubtitleCount int      `json:"subtitle_count,omitempty"`
	SubtitleLangs []string `json:"subtitle_languages,omitempty"`

	// File information
	Filename string `json:"filename,omitempty"`
	FilePath string `json:"file_path,omitempty"`
}

// generateVideoMetadata extracts all metadata from video file using ffprobe and returns as JSON
func generateVideoMetadata(input *media.Request) error {
	// Generate cache key for metadata
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(input.OriginalFilePath+"_metadata")))
	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "video_metadata")
	os.MkdirAll(cacheDir, 0755)

	jsonPath := filepath.Join(cacheDir, fmt.Sprintf("%s.json", cacheKey))

	// Check if cached version exists
	if _, err := os.Stat(jsonPath); err == nil {
		if input.Debug {
			log.Debug("Cache hit for video metadata", "trace_id", input.TraceID, "cache_key", cacheKey, "json_path", jsonPath)
			input.Request.Set("X-Debug-Video-Metadata-Cache-Status", "HIT")
			input.Request.Set("X-Debug-Video-Metadata-Cache-Key", cacheKey)
			input.Request.Set("X-Debug-Video-Metadata-Cache-Path", jsonPath)
		}
		input.ProcessedFilePath = jsonPath
		input.ProcessedMimeType = "application/json"
		return nil
	}

	if input.Debug {
		log.Debug("Cache miss for video metadata", "trace_id", input.TraceID, "cache_key", cacheKey, "json_path", jsonPath)
		input.Request.Set("X-Debug-Video-Metadata-Cache-Status", "MISS")
		input.Request.Set("X-Debug-Video-Metadata-Cache-Key", cacheKey)
		input.Request.Set("X-Debug-Video-Metadata-Cache-Path", jsonPath)
	}

	// Get file info for size
	fileInfo, err := os.Stat(input.StagedFilePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Initialize metadata structure
	metadata := VideoMetadata{
		Filename: filepath.Base(input.OriginalFilePath),
		FilePath: input.OriginalFilePath,
		Size:     fileInfo.Size(),
	}

	// Get video duration
	duration, err := getVideoDuration(input.StagedFilePath)
	if err != nil {
		log.Debug("Failed to get video duration", "trace_id", input.TraceID, "error", err)
	} else {
		metadata.Duration = duration
	}

	// Get detailed video information using a single ffprobe call for both
	// format and stream data, avoiding a second process spawn.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	probeCmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		input.StagedFilePath)

	probeOutput, err := probeCmd.Output()
	if err != nil {
		log.Debug("Failed to get video information", "trace_id", input.TraceID, "error", err)
	} else {
		var probeData map[string]interface{}
		if err := json.Unmarshal(probeOutput, &probeData); err == nil {
			// Parse format section
			if format, ok := probeData["format"].(map[string]interface{}); ok {
				if formatName, ok := format["format_name"].(string); ok {
					metadata.Format = formatName
				}
				if bitrate, ok := format["bit_rate"].(string); ok {
					bitrateInt, _ := strconv.Atoi(bitrate)
					metadata.Bitrate = bitrateInt
				}
			}
			// Parse streams section
			if streams, ok := probeData["streams"].([]interface{}); ok {
				var subtitleCount int
				var subtitleLangs []string

				for _, stream := range streams {
					if streamMap, ok := stream.(map[string]interface{}); ok {
						codecType, _ := streamMap["codec_type"].(string)

						switch codecType {
						case "video":
							if codec, ok := streamMap["codec_name"].(string); ok {
								metadata.VideoCodec = codec
							}
							if width, ok := streamMap["width"].(float64); ok {
								metadata.Width = int(width)
							}
							if height, ok := streamMap["height"].(float64); ok {
								metadata.Height = int(height)
							}
							if width, ok := streamMap["width"].(float64); ok {
								if height, ok := streamMap["height"].(float64); ok {
									metadata.AspectRatio = media.GetAspectRatioName(width, height)
								}
							}
							if colorSpace, ok := streamMap["color_space"].(string); ok {
								metadata.ColorSpace = colorSpace
							}
							if pixFmt, ok := streamMap["pix_fmt"].(string); ok {
								metadata.PixelFormat = pixFmt
							}

							// Extract frame rate
							if rFrameRate, ok := streamMap["r_frame_rate"].(string); ok {
								parts := strings.Split(rFrameRate, "/")
								if len(parts) == 2 {
									num, _ := strconv.ParseFloat(parts[0], 64)
									den, _ := strconv.ParseFloat(parts[1], 64)
									if den > 0 {
										metadata.FrameRate = num / den
									}
								}
							}

						case "audio":
							if codec, ok := streamMap["codec_name"].(string); ok {
								metadata.AudioCodec = codec
							}
							if channels, ok := streamMap["channels"].(float64); ok {
								metadata.AudioChannels = int(channels)
							}
							if sampleRate, ok := streamMap["sample_rate"].(string); ok {
								sampleRateInt, _ := strconv.Atoi(sampleRate)
								metadata.SampleRate = sampleRateInt
							}

						case "subtitle":
							subtitleCount++
							if tags, ok := streamMap["tags"].(map[string]interface{}); ok {
								if language, ok := tags["language"].(string); ok {
									subtitleLangs = append(subtitleLangs, language)
								}
							}
						}
					}
				}

				metadata.SubtitleCount = subtitleCount
				metadata.SubtitleLangs = subtitleLangs
			}
		}
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %v", err)
	}

	// Write JSON to file
	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON metadata file: %v", err)
	}

	// Set the processed file path and MIME type
	input.ProcessedFilePath = jsonPath
	input.ProcessedMimeType = "application/json"

	return nil
}

// processVideo handles both preview and thumbnail generation
func processVideo(input *media.Request) error {
	if input.Debug {
		log.Debug("Starting video processing", "trace_id", input.TraceID, "preview", input.Options.Preview, "thumbnail", input.Options.Thumbnail, "detail", input.Options.Detail)
	}

	// Check if this is a detail request (JSON metadata output)
	if input.Options.Detail {
		if input.Debug {
			log.Debug("Processing video metadata", "trace_id", input.TraceID)
		}
		return generateVideoMetadata(input)
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
