package media

var (
	ImageSizes = []int{
		3840, // 4K UHD
		2560, // QHD (1440p)
		1920, // Full HD (1080p)
		1600, // HD+ / UXGA
		1280, // HD (720p)
		1024,
		960, // SD+
		854, // 480p (16:9)
		800, // SVGA
		720, // HD (alternative)
		640, // VGA
		512,
		480, // nHD
		360, // LD
		320, // QVGA
		240, // QQVGA
		160, // HQVGA
		128, // Tiny thumbnails
		96,  // Very small
		64,  // Icons
		32,  // Very small icons
	}

	ImageQuality = []int{
		100,
		90,
		85,
		80,
		75,
		60,
		50,
	}
)
