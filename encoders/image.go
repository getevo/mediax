package encoders

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getevo/evo/v2/lib/gpath"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/rwcarlsen/goexif/exif"
	"mediax/apps/media"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var Png = media.Encoder{
	Mime:      "image/png",
	Processor: Imagick,
}

var Jpeg = media.Encoder{
	Mime:      "image/jpeg",
	Processor: Imagick,
}

var Gif = media.Encoder{
	Mime:      "image/gif",
	Processor: Imagick,
}

var Webp = media.Encoder{
	Mime:      "image/webp",
	Processor: Imagick,
}

var Avif = media.Encoder{
	Mime:      "image/avif",
	Processor: Imagick,
}

// ExtractImageExif extracts metadata from an image file using both ImageMagick and EXIF
func ExtractImageExif(input *media.Request) (map[string]interface{}, error) {
	var metadata = map[string]interface{}{}
	absPath := input.StagedFilePath
	if absPath == "" {
		return nil, fmt.Errorf("staged file path is empty")
	}

	// First, extract metadata using ImageMagick
	if input.Debug {
		log.Debug("Using ImageMagick to extract metadata", "trace_id", input.TraceID, "path", absPath)
	}

	imageMagickMetadata, err := extractImageMagickMetadata(absPath)
	if err != nil {
		if input.Debug {
			log.Error("Error extracting ImageMagick metadata", "trace_id", input.TraceID, "error", err.Error())
		}
		// Continue with EXIF extraction even if ImageMagick failed
	} else {
		// Add ImageMagick metadata to our result
		for key, value := range imageMagickMetadata {
			metadata[key] = value
		}
	}

	// Now, try to extract EXIF data to add more details or override existing keys
	f, err := os.Open(absPath)
	if err != nil {
		// If we can't open the file for EXIF, return whatever metadata we have from ImageMagick
		if input.Debug {
			log.Error("Cannot open image for EXIF extraction", "trace_id", input.TraceID, "error", err.Error())
		}
		return metadata, nil
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		// Some images may not have EXIF data, which is not an error
		// Return whatever metadata we have from ImageMagick
		if input.Debug && !strings.Contains(err.Error(), "exif") {
			log.Error("Failed to decode EXIF", "trace_id", input.TraceID, "error", err.Error())
		}
		return metadata, nil
	}

	// EXIF data is available, extract it and add/overwrite to metadata
	// Common EXIF tags to extract
	tags := []string{
		"Make", "Model", "Software", "LensModel",
		"DateTime", "DateTimeOriginal", "SubSecTimeOriginal",
		"ExposureTime", "FNumber", "ISOSpeedRatings",
		"ShutterSpeedValue", "ApertureValue", "FocalLength",
		"Orientation", "WhiteBalance", "Flash",
		"PixelXDimension", "PixelYDimension",
		"XResolution", "YResolution", "ResolutionUnit",
	}

	exifVals := make(map[string]string)

	for _, tag := range tags {
		if val, err := x.Get(exif.FieldName(tag)); err == nil {
			if valStr, err := val.StringVal(); err == nil {
				exifVals[tag] = valStr
				metadata[strings.ToLower(tag)] = valStr
			}
		}
	}

	// Width, height from EXIF (overwrite ImageMagick values if available)
	widthStr := exifVals["PixelXDimension"]
	heightStr := exifVals["PixelYDimension"]

	if widthStr != "" && heightStr != "" {
		metadata["width"] = widthStr
		metadata["height"] = heightStr
		// Aspect ratio
		var width, height float64
		fmt.Sscanf(widthStr, "%f", &width)
		fmt.Sscanf(heightStr, "%f", &height)
		if width > 0 && height > 0 {
			metadata["aspect_ratio"] = media.GetAspectRatioName(width, height)
		}
	}

	// DPI calculation from EXIF (overwrite ImageMagick values if available)
	dpiX := exifVals["XResolution"]
	dpiY := exifVals["YResolution"]
	resUnit := strings.ToLower(exifVals["ResolutionUnit"]) // 2 = inches, 3 = cm

	if dpiX != "" && resUnit == "2" {
		metadata["dpi_x"] = dpiX
	}
	if dpiY != "" && resUnit == "2" {
		metadata["dpi_y"] = dpiY
	}

	// GPS from EXIF
	if lat, long, err := x.LatLong(); err == nil {
		metadata["latitude"] = lat
		metadata["longitude"] = long
	}

	return metadata, nil
}

// processImage handles image processing operations
func processImage(input *media.Request) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}

	// Extract metadata if detail=true
	if input.Options.Detail {
		// Generate metadata cache file path
		metadataCacheFile := strings.TrimSuffix(input.StagedFilePath, filepath.Ext(input.StagedFilePath)) + ".metadata.json"

		// Check if metadata cache file exists
		if gpath.IsFileExist(metadataCacheFile) {
			// Check if the original file has been modified since the cache was created
			originalFileInfo, originalErr := os.Stat(input.StagedFilePath)
			cacheFileInfo, cacheErr := os.Stat(metadataCacheFile)

			// Only use cache if both files exist and the original file is not newer than the cache
			if originalErr == nil && cacheErr == nil && !originalFileInfo.ModTime().After(cacheFileInfo.ModTime()) {
				// Read metadata from cache file
				if input.Debug {
					log.Debug("Reading metadata from cache", "trace_id", input.TraceID, "cache_file", metadataCacheFile)
				}

				cachedData, err := os.ReadFile(metadataCacheFile)
				if err == nil {
					// Deserialize metadata
					var metadata map[string]interface{}
					if err := json.Unmarshal(cachedData, &metadata); err == nil {
						input.Metadata = metadata
						if input.Debug {
							log.Debug("Metadata loaded from cache", "trace_id", input.TraceID)
						}
					} else if input.Debug {
						log.Error("Error deserializing cached metadata", "trace_id", input.TraceID, "error", err.Error())
					}
				} else if input.Debug {
					log.Error("Error reading metadata cache file", "trace_id", input.TraceID, "error", err.Error())
				}
			} else if input.Debug {
				if originalErr != nil {
					log.Error("Error checking original file", "trace_id", input.TraceID, "error", originalErr.Error())
				} else if cacheErr != nil {
					log.Error("Error checking cache file", "trace_id", input.TraceID, "error", cacheErr.Error())
				} else {
					log.Debug("Cache invalidated - original file is newer", "trace_id", input.TraceID)
				}
			}
		}

		// If metadata wasn't loaded from cache, extract it
		if input.Metadata == nil {
			metadata, err := ExtractImageExif(input)
			if err != nil {
				// Log the error but continue with image processing
				if input.Debug {
					log.Error("Error extracting EXIF data", "trace_id", input.TraceID, "error", err.Error())
				}
			} else {
				input.Metadata = metadata

				// Add basic file info if not already present
				fileInfo, err := os.Stat(input.StagedFilePath)
				if err == nil {
					// Add file size
					input.Metadata["file_size"] = fileInfo.Size()
					input.Metadata["modified_time"] = fileInfo.ModTime().Format(time.RFC3339)
				}

				// Cache the metadata
				if input.Metadata != nil {
					if input.Debug {
						log.Debug("Caching metadata", "trace_id", input.TraceID, "cache_file", metadataCacheFile)
					}

					// Serialize metadata
					cachedData, err := json.Marshal(input.Metadata)
					if err == nil {
						// Write to cache file
						err = os.WriteFile(metadataCacheFile, cachedData, 0600)
						if err != nil && input.Debug {
							log.Error("Error writing metadata cache file", "trace_id", input.TraceID, "error", err.Error())
						}
					} else if input.Debug {
						log.Error("Error serializing metadata", "trace_id", input.TraceID, "error", err.Error())
					}
				}
			}
		}
	}

	// Currently we only have one type of image processing
	// If more types are added in the future, we can add conditional logic here
	return convertImage(input)
}

// convertImage handles the standard image conversion using ImageMagick
func convertImage(input *media.Request) error {
	var opts = input.Options
	input.ProcessedFilePath = strings.TrimSuffix(input.StagedFilePath, filepath.Ext(input.StagedFilePath)) + opts.ToString() + "." + opts.OutputFormat

	if gpath.IsFileExist(input.ProcessedFilePath) {
		return nil
	}
	args := []string{input.StagedFilePath}

	// Handle resizing logic
	var resizeStr string
	if opts.Width == 0 && opts.Height == 0 {
		// No resize
	} else if opts.KeepAspectRatio {
		// Keep aspect ratio
		if opts.Width == 0 {
			resizeStr = fmt.Sprintf("x%d", opts.Height)
		} else if opts.Height == 0 {
			resizeStr = fmt.Sprintf("%d", opts.Width)
		} else {
			resizeStr = fmt.Sprintf("%dx%d", opts.Width, opts.Height)
		}
		args = append(args, "-resize", resizeStr)
	} else {
		// Resize to fill and crop later
		if opts.Width == 0 || opts.Height == 0 {
			// Can't crop without both dimensions
			resizeStr = fmt.Sprintf("%dx%d", opts.Width, opts.Height)
			args = append(args, "-resize", resizeStr)
		} else {
			resizeStr = fmt.Sprintf("%dx%d^", opts.Width, opts.Height)
			args = append(args, "-resize", resizeStr)
			args = append(args,
				"-gravity", getGravity(opts.CropDirection),
				"-crop", fmt.Sprintf("%dx%d+0+0", opts.Width, opts.Height),
				//"+repage",
			)
		}
	}

	// Apply quality if specified
	if opts.Quality > 0 {
		args = append(args, "-quality", fmt.Sprintf("%d", opts.Quality))
	}

	args = append(args, input.ProcessedFilePath)
	ctx, cancel := context.WithTimeout(context.Background(), imageConvertTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "convert", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("convert timed out after %s", imageConvertTimeout)
		}
		return fmt.Errorf("convert error: %v\noutput: %s", err, truncateOutput(output))
	}

	return nil
}

// Imagick processor for image conversion
var Imagick = processImage

// Map crop direction to ImageMagick gravity
func getGravity(direction string) string {
	switch strings.ToLower(direction) {
	case "top":
		return "north"
	case "bottom":
		return "south"
	case "left":
		return "west"
	case "right":
		return "east"
	default:
		return "center"
	}
}

// extractImageMagickMetadata extracts image metadata using ImageMagick's identify command
func extractImageMagickMetadata(filePath string) (map[string]interface{}, error) {
	metadata := make(map[string]interface{})

	// Run identify command with detailed format
	ctx, cancel := context.WithTimeout(context.Background(), imageConvertTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "identify", "-format", "%w,%h,%[colorspace],%[depth],%[quality],%[format],%[exif:*]", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("imagemagick identify timed out after %s", imageConvertTimeout)
		}
		return nil, fmt.Errorf("imagemagick identify error: %v\noutput: %s", err, truncateOutput(output))
	}

	// Parse the output
	parts := strings.Split(string(output), ",")
	if len(parts) >= 2 {
		// Extract width and height
		if width, err := strconv.ParseFloat(parts[0], 64); err == nil {
			metadata["width"] = fmt.Sprintf("%g", width)

			// Extract height
			if height, err := strconv.ParseFloat(parts[1], 64); err == nil {
				metadata["height"] = fmt.Sprintf("%g", height)

				// Calculate aspect ratio
				if width > 0 && height > 0 {
					metadata["aspect_ratio"] = media.GetAspectRatioName(width, height)
				}
			}
		}

		// Extract additional metadata if available
		if len(parts) >= 6 {
			if parts[2] != "" {
				metadata["colorspace"] = parts[2]
			}
			if parts[3] != "" {
				metadata["bit_depth"] = parts[3]
			}
			if parts[4] != "" {
				metadata["quality"] = parts[4]
			}
		}

		// Extract any EXIF data that might be available through ImageMagick
		if len(parts) > 6 {
			for i := 6; i < len(parts); i++ {
				if keyValue := strings.SplitN(parts[i], "=", 2); len(keyValue) == 2 {
					key := strings.ToLower(keyValue[0])
					metadata[key] = keyValue[1]
				}
			}
		}
	}

	// Get more detailed information using verbose mode
	ctx2, cancel2 := context.WithTimeout(context.Background(), imageConvertTimeout)
	defer cancel2()
	cmd = exec.CommandContext(ctx2, "identify", "-verbose", filePath)
	verboseOutput, err := cmd.CombinedOutput()
	if err == nil {
		// Extract DPI information using regex
		dpiRegex := regexp.MustCompile(`Resolution: (\d+)x(\d+)`)
		if matches := dpiRegex.FindStringSubmatch(string(verboseOutput)); len(matches) == 3 {
			metadata["dpi_x"] = matches[1]
			metadata["dpi_y"] = matches[2]
		}

		// Extract color profile information
		if strings.Contains(string(verboseOutput), "Profile-icc:") {
			metadata["has_color_profile"] = true
		}

		// Extract transparency information
		if strings.Contains(string(verboseOutput), "Alpha:") {
			metadata["has_transparency"] = true
		}
	}

	return metadata, nil
}
