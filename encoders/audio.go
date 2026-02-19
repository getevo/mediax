package encoders

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/dhowden/tag"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"mediax/apps/media"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Audio encoders with conversion support using FFmpeg
var Mp3 = media.Encoder{
	Mime:      "audio/mpeg",
	Processor: FFmpeg,
}

var Wav = media.Encoder{
	Mime:      "audio/wav",
	Processor: FFmpeg,
}

var Flac = media.Encoder{
	Mime:      "audio/flac",
	Processor: FFmpeg,
}

var Aac = media.Encoder{
	Mime:      "audio/aac",
	Processor: FFmpeg,
}

var Ogg = media.Encoder{
	Mime:      "audio/ogg",
	Processor: FFmpeg,
}

var M4a = media.Encoder{
	Mime:      "audio/mp4",
	Processor: FFmpeg,
}

var Wma = media.Encoder{
	Mime:      "audio/x-ms-wma",
	Processor: FFmpeg,
}

var Opus = media.Encoder{
	Mime:      "audio/opus",
	Processor: FFmpeg,
}

// Audio encoders for serving without conversion (like video encoders)
var Mp3Direct = media.Encoder{
	Mime:      "audio/mpeg",
	Processor: nil, // No processing needed
}

var WavDirect = media.Encoder{
	Mime:      "audio/wav",
	Processor: nil, // No processing needed
}

var FlacDirect = media.Encoder{
	Mime:      "audio/flac",
	Processor: nil, // No processing needed
}

var AacDirect = media.Encoder{
	Mime:      "audio/aac",
	Processor: nil, // No processing needed
}

var OggDirect = media.Encoder{
	Mime:      "audio/ogg",
	Processor: nil, // No processing needed
}

var M4aDirect = media.Encoder{
	Mime:      "audio/mp4",
	Processor: nil, // No processing needed
}

var WmaDirect = media.Encoder{
	Mime:      "audio/x-ms-wma",
	Processor: nil, // No processing needed
}

var OpusDirect = media.Encoder{
	Mime:      "audio/opus",
	Processor: nil, // No processing needed
}

// AudioMetadata represents all audio metadata information
type AudioMetadata struct {
	// Basic metadata
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Composer    string `json:"composer,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Year        int    `json:"year,omitempty"`
	Track       int    `json:"track,omitempty"`
	TrackTotal  int    `json:"track_total,omitempty"`
	Disc        int    `json:"disc,omitempty"`
	DiscTotal   int    `json:"disc_total,omitempty"`

	// Technical metadata
	Format   string `json:"format,omitempty"`
	FileType string `json:"file_type,omitempty"`

	// File information
	Filename string `json:"filename,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`

	// Artwork information
	HasArtwork  bool `json:"has_artwork"`
	ArtworkSize int  `json:"artwork_size,omitempty"`
}

// generateAudioMetadata extracts all metadata from audio file and returns as JSON
func generateAudioMetadata(input *media.Request) error {
	// Open the audio file
	file, err := os.Open(input.StagedFilePath)
	if err != nil {
		return fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Parse metadata using tag library
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("failed to read audio metadata: %v", err)
	}

	// Create metadata structure
	audioMeta := AudioMetadata{
		Title:       metadata.Title(),
		Artist:      metadata.Artist(),
		Album:       metadata.Album(),
		AlbumArtist: metadata.AlbumArtist(),
		Composer:    metadata.Composer(),
		Genre:       metadata.Genre(),
		Year:        metadata.Year(),
		Format:      string(metadata.Format()),
		FileType:    string(metadata.FileType()),
		Filename:    filepath.Base(input.OriginalFilePath),
		FileSize:    fileInfo.Size(),
		HasArtwork:  metadata.Picture() != nil,
	}

	// Get track and disc numbers
	track, trackTotal := metadata.Track()
	audioMeta.Track = track
	audioMeta.TrackTotal = trackTotal

	disc, discTotal := metadata.Disc()
	audioMeta.Disc = disc
	audioMeta.DiscTotal = discTotal

	// Get artwork size if available
	if picture := metadata.Picture(); picture != nil {
		audioMeta.ArtworkSize = len(picture.Data)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(audioMeta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %v", err)
	}

	// Create a temporary JSON file
	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "audio_metadata")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create audio metadata cache dir: %w", err)
	}

	// Generate cache key for metadata
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(input.OriginalFilePath+"_metadata")))
	jsonPath := filepath.Join(cacheDir, fmt.Sprintf("%s.json", cacheKey))

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

// generateAudioThumbnail creates a thumbnail from audio file's embedded artwork using tag library
func generateAudioThumbnail(input *media.Request) error {
	if input.Options.Thumbnail == "" {
		return nil
	}

	// Determine output format (default to jpeg)
	outputFormat := input.Options.OutputFormat
	if outputFormat == "" {
		outputFormat = "jpeg"
	}

	// Generate cache key and check if thumbnail already exists
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(input.OriginalFilePath+input.Options.Thumbnail+outputFormat)))
	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "audio_thumbnails")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create audio thumbnail cache dir: %w", err)
	}

	// Determine final file extension
	_, finalExtension := getImageFormat(outputFormat)
	finalPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.%s", cacheKey, input.Options.Thumbnail, finalExtension))

	// Check if cached version exists
	if _, err := os.Stat(finalPath); err == nil {
		if input.Debug {
			log.Debug("Cache hit for audio thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
			input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Status", "HIT")
			input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Key", cacheKey)
			input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Path", finalPath)
		}
		input.ProcessedFilePath = finalPath
		input.ProcessedMimeType = getImageMimeType(outputFormat)
		return nil
	}

	if input.Debug {
		log.Debug("Cache miss for audio thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
		input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Status", "MISS")
		input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Key", cacheKey)
		input.Request.Set("X-Debug-Audio-Thumbnail-Cache-Path", finalPath)
	}

	// Open the audio file
	file, err := os.Open(input.StagedFilePath)
	if err != nil {
		return fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	// Parse metadata using tag library
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("failed to read audio metadata: %v", err)
	}

	// Get embedded artwork
	picture := metadata.Picture()
	if picture == nil {
		return fmt.Errorf("audio file does not contain embedded artwork")
	}

	// Step 1: Save the embedded artwork as a temporary JPEG file
	jpegPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_temp.jpg", cacheKey, input.Options.Thumbnail))

	err = os.WriteFile(jpegPath, picture.Data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save embedded artwork: %v", err)
	}

	// Step 2: Use ImageMagick convert to change format and size based on user input
	args := []string{jpegPath}

	// Parse thumbnail parameter for size
	if strings.Contains(input.Options.Thumbnail, "x") {
		// Custom dimensions (e.g., "256x256")
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
		if rmErr := os.Remove(jpegPath); rmErr != nil && !os.IsNotExist(rmErr) {
			log.Warning("failed to remove temp jpeg", "path", jpegPath, "error", rmErr)
		}
		return fmt.Errorf("ImageMagick convert error: %v\noutput: %s", err, output)
	}

	// Clean up temporary JPEG file
	if rmErr := os.Remove(jpegPath); rmErr != nil && !os.IsNotExist(rmErr) {
		log.Warning("failed to remove temp jpeg", "path", jpegPath, "error", rmErr)
	}

	input.ProcessedFilePath = finalPath
	input.ProcessedMimeType = getImageMimeType(outputFormat)
	return nil
}

// processAudio handles different audio processing operations based on request type
func processAudio(input *media.Request) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	var opts = input.Options

	// Check if this is a detail request (JSON metadata output)
	if opts.Detail {
		return generateAudioMetadata(input)
	}

	// Check if this is a thumbnail request (image format output)
	if opts.Thumbnail != "" {
		if opts.OutputFormat == "" {
			opts.OutputFormat = "jpg"
		}
		if !isImageFormat(opts.OutputFormat) {
			opts.OutputFormat = "jpg"
		}
		return generateAudioThumbnail(input)
	}

	// Standard audio conversion
	return convertAudio(input)
}

// convertAudio handles the standard audio conversion using FFmpeg
func convertAudio(input *media.Request) error {
	var opts = input.Options
	input.ProcessedFilePath = strings.TrimSuffix(input.StagedFilePath, filepath.Ext(input.StagedFilePath)) + opts.ToString() + "." + opts.OutputFormat

	if gpath.IsFileExist(input.ProcessedFilePath) {
		return nil
	}

	args := []string{"-i", input.StagedFilePath}

	// Audio quality settings
	if opts.Quality > 0 {
		switch strings.ToLower(opts.OutputFormat) {
		case "mp3":
			// For MP3, quality ranges from 0 (best) to 9 (worst)
			// Convert our 1-100 scale to 0-9 scale (inverted)
			mp3Quality := 9 - (opts.Quality * 9 / 100)
			args = append(args, "-q:a", fmt.Sprintf("%d", mp3Quality))
		case "aac", "m4a":
			// For AAC, use bitrate based on quality (64k to 320k)
			bitrate := 64 + (opts.Quality * 256 / 100)
			args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
		case "ogg":
			// For OGG, quality ranges from -1 to 10
			oggQuality := -1 + (opts.Quality * 11 / 100)
			args = append(args, "-q:a", fmt.Sprintf("%d", oggQuality))
		case "opus":
			// For Opus, use bitrate (32k to 512k)
			bitrate := 32 + (opts.Quality * 480 / 100)
			args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
		default:
			// For other formats, use generic bitrate
			bitrate := 64 + (opts.Quality * 256 / 100)
			args = append(args, "-b:a", fmt.Sprintf("%dk", bitrate))
		}
	}

	// Audio codec selection based on output format
	switch strings.ToLower(opts.OutputFormat) {
	case "mp3":
		args = append(args, "-codec:a", "libmp3lame")
	case "aac", "m4a":
		args = append(args, "-codec:a", "aac")
	case "ogg":
		args = append(args, "-codec:a", "libvorbis")
	case "flac":
		args = append(args, "-codec:a", "flac")
	case "wav":
		args = append(args, "-codec:a", "pcm_s16le")
	case "wma":
		args = append(args, "-codec:a", "wmav2")
	case "opus":
		args = append(args, "-codec:a", "libopus")
	}

	// Overwrite output file if it exists
	args = append(args, "-y")

	// Add output file
	args = append(args, input.ProcessedFilePath)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %v\noutput: %s", err, output)
	}

	return nil
}

// FFmpeg processor for audio conversion
var FFmpeg = processAudio

// Helper functions for audio thumbnail processing
func isImageFormat(format string) bool {
	switch strings.ToLower(format) {
	case "jpg", "jpeg", "png", "webp", "avif", "gif":
		return true
	default:
		return false
	}
}
