package encoders

import (
	"crypto/md5"
	"fmt"
	"github.com/getevo/evo/v2/lib/log"
	"mediax/apps/media"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Document encoders for various document formats

// Pdf document format
var Pdf = media.Encoder{
	Mime:      "application/pdf",
	Processor: processDocument,
}

// Docx Microsoft Office formats
var Docx = media.Encoder{
	Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	Processor: processDocument,
}

var Xlsx = media.Encoder{
	Mime:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	Processor: processDocument,
}

var Pptx = media.Encoder{
	Mime:      "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	Processor: processDocument,
}

// Legacy Microsoft Office formats
var Doc = media.Encoder{
	Mime:      "application/msword",
	Processor: processDocument,
}

var Xls = media.Encoder{
	Mime:      "application/vnd.ms-excel",
	Processor: processDocument,
}

var Ppt = media.Encoder{
	Mime:      "application/vnd.ms-powerpoint",
	Processor: processDocument,
}

// OpenDocument formats
var Odt = media.Encoder{
	Mime:      "application/vnd.oasis.opendocument.text",
	Processor: processDocument,
}

var Ods = media.Encoder{
	Mime:      "application/vnd.oasis.opendocument.spreadsheet",
	Processor: processDocument,
}

var Odp = media.Encoder{
	Mime:      "application/vnd.oasis.opendocument.presentation",
	Processor: processDocument,
}

// Text formats
var Txt = media.Encoder{
	Mime:      "text/plain",
	Processor: processDocument,
}

var Rtf = media.Encoder{
	Mime:      "application/rtf",
	Processor: processDocument,
}

var Csv = media.Encoder{
	Mime:      "text/csv",
	Processor: processDocument,
}

// Other common formats
var Epub = media.Encoder{
	Mime:      "application/epub+zip",
	Processor: processDocument,
}

var Xml = media.Encoder{
	Mime:      "application/xml",
	Processor: processDocument,
}

var Json = media.Encoder{
	Mime:      "application/json",
	Processor: processDocument,
}

// DocumentMetadata represents document metadata information
type DocumentMetadata struct {
	// File information
	Filename     string `json:"filename,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	FileType     string `json:"file_type,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

// generateDocumentThumbnail creates a thumbnail from the first page of a document
func generateDocumentThumbnail(input *media.Request) error {
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
	cacheDir := filepath.Join(input.Origin.Project.CacheDir, "document_thumbnails")
	os.MkdirAll(cacheDir, 0755)

	// Determine final file extension
	_, finalExtension := getImageFormat(outputFormat)
	finalPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s.%s", cacheKey, input.Options.Thumbnail, finalExtension))

	// Check if cached version exists
	if _, err := os.Stat(finalPath); err == nil {
		if input.Debug {
			log.Debug("Cache hit for document thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
			input.Request.Set("X-Debug-Document-Thumbnail-Cache-Status", "HIT")
			input.Request.Set("X-Debug-Document-Thumbnail-Cache-Key", cacheKey)
			input.Request.Set("X-Debug-Document-Thumbnail-Cache-Path", finalPath)
		}
		input.ProcessedFilePath = finalPath
		input.ProcessedMimeType = getImageMimeType(outputFormat)
		return nil
	}

	if input.Debug {
		log.Debug("Cache miss for document thumbnail", "trace_id", input.TraceID, "cache_key", cacheKey, "thumbnail", input.Options.Thumbnail, "final_path", finalPath)
		input.Request.Set("X-Debug-Document-Thumbnail-Cache-Status", "MISS")
		input.Request.Set("X-Debug-Document-Thumbnail-Cache-Key", cacheKey)
		input.Request.Set("X-Debug-Document-Thumbnail-Cache-Path", finalPath)
	}

	// Step 1: Convert first page to image using appropriate tool based on file type
	tempImagePath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_temp.png", cacheKey, input.Options.Thumbnail))
	genericThumbnailPath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_generic.png", cacheKey, input.Options.Thumbnail))

	var conversionSuccessful bool
	fileExt := strings.ToLower(filepath.Ext(input.StagedFilePath))

	// Try to convert document to image
	switch {
	case fileExt == ".pdf":
		// Use pdftoppm for PDF files
		if err := convertPdfToImage(input.StagedFilePath, tempImagePath); err == nil {
			conversionSuccessful = true
		} else if input.Debug {
			log.Debug("PDF to image conversion failed, will use generic thumbnail", "trace_id", input.TraceID, "error", err.Error())
		}
	case fileExt == ".docx" || fileExt == ".doc" || fileExt == ".odt" ||
		fileExt == ".xlsx" || fileExt == ".xls" || fileExt == ".ods" ||
		fileExt == ".pptx" || fileExt == ".ppt" || fileExt == ".odp":
		// Use LibreOffice for Office documents
		if err := convertOfficeToImage(input.StagedFilePath, tempImagePath); err == nil {
			conversionSuccessful = true
		} else if input.Debug {
			log.Debug("Office to image conversion failed, will use generic thumbnail", "trace_id", input.TraceID, "error", err.Error())
		}
	}

	// If conversion failed, create a generic thumbnail
	if !conversionSuccessful {
		if err := createGenericThumbnail(input.StagedFilePath, genericThumbnailPath, filepath.Ext(input.StagedFilePath)[1:]); err == nil {
			tempImagePath = genericThumbnailPath
			conversionSuccessful = true
		} else if input.Debug {
			log.Debug("Generic thumbnail creation failed", "trace_id", input.TraceID, "error", err.Error())
		}
	}

	// Step 2: Use ImageMagick convert to resize and format the thumbnail
	// Always run convert command to generate proper image, even if initial conversion failed
	// If all conversions failed, create a blank image
	var sourceImage string
	if conversionSuccessful {
		sourceImage = tempImagePath
	} else {
		// Create a blank image as a last resort
		blankImagePath := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_blank.png", cacheKey, input.Options.Thumbnail))
		err := exec.Command("convert", "-size", "800x600", "xc:white",
			"-gravity", "center",
			"-pointsize", "24",
			"-annotate", "0", "Document Preview Unavailable",
			blankImagePath).Run()
		if err != nil {
			return fmt.Errorf("failed to create blank image: %v", err)
		}
		sourceImage = blankImagePath
		defer os.Remove(blankImagePath)
	}

	args := []string{sourceImage}

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
		// Clean up temporary files
		if conversionSuccessful {
			os.Remove(tempImagePath)
			if tempImagePath != genericThumbnailPath {
				os.Remove(genericThumbnailPath)
			}
		}
		return fmt.Errorf("ImageMagick convert error: %v\noutput: %s", err, output)
	}

	// Clean up temporary files
	if conversionSuccessful {
		os.Remove(tempImagePath)
		if tempImagePath != genericThumbnailPath {
			os.Remove(genericThumbnailPath)
		}
	}

	input.ProcessedFilePath = finalPath
	input.ProcessedMimeType = getImageMimeType(outputFormat)
	return nil
}

// convertPdfToImage converts the first page of a PDF to an image
func convertPdfToImage(pdfPath, outputPath string) error {
	cmd := exec.Command("pdftoppm", "-png", "-singlefile", "-f", "1", "-l", "1", pdfPath, strings.TrimSuffix(outputPath, ".png"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pdftoppm error: %v\noutput: %s", err, output)
	}

	// pdftoppm adds "-1" to the filename, so we need to rename it
	generatedPath := strings.TrimSuffix(outputPath, ".png") + "-1.png"
	if _, err := os.Stat(generatedPath); err == nil {
		err = os.Rename(generatedPath, outputPath)
		if err != nil {
			return fmt.Errorf("failed to rename generated image: %v", err)
		}
	}

	return nil
}

// convertOfficeToImage converts the first page of an Office document to an image
func convertOfficeToImage(officePath, outputPath string) error {
	// Create a temporary directory for conversion
	tempDir := filepath.Join(filepath.Dir(outputPath), "temp_"+filepath.Base(officePath))
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// Use LibreOffice to convert to PDF first
	// LibreOffice will create a PDF with the same base name as the input file
	baseFileName := filepath.Base(officePath)
	baseNameWithoutExt := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))
	expectedPdfPath := filepath.Join(tempDir, baseNameWithoutExt+".pdf")

	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", tempDir, officePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("LibreOffice conversion error: %v\noutput: %s", err, output)
	}

	// Check if the PDF was created
	if _, err := os.Stat(expectedPdfPath); os.IsNotExist(err) {
		return fmt.Errorf("LibreOffice did not create the expected PDF file at %s", expectedPdfPath)
	}

	// Now convert the PDF to image using pdftoppm
	return convertPdfToImage(expectedPdfPath, outputPath)
}

// createGenericThumbnail creates a generic thumbnail for document types without specific converters
func createGenericThumbnail(docPath, outputPath, fileType string) error {
	// Create a blank canvas with file type text
	cmd := exec.Command("convert", "-size", "800x600", "xc:white",
		"-gravity", "center",
		"-pointsize", "72",
		"-annotate", "0", fileType,
		outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick error: %v\noutput: %s", err, output)
	}
	return nil
}

// processDocument handles different document processing operations
func processDocument(input *media.Request) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}

	if !isImageFormat(input.Options.OutputFormat) {
		input.Options.OutputFormat = "jpg"
	}

	return generateDocumentThumbnail(input)
}
