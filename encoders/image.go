package encoders

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/gpath"
	"mediax/apps/media"
	"os/exec"
	"path/filepath"
	"strings"
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

var Imagick = func(input *media.Request) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
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
	cmd := exec.Command("convert", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert error: %v\noutput: %s", err, output)
	}

	return nil
}

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
