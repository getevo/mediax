package encoders

import (
	"mediax/apps/media"
)

// MediaTypes defines all supported media types with their encoders
var MediaTypes = map[string]*media.Type{
	// Image formats
	"jpg": {
		Extension: "jpg",
		Mime:      "image/jpeg",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},
	"jpeg": {
		Extension: "jpeg",
		Mime:      "image/jpeg",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},
	"png": {
		Extension: "png",
		Mime:      "image/png",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},
	"gif": {
		Extension: "gif",
		Mime:      "image/gif",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},
	"webp": {
		Extension: "webp",
		Mime:      "image/webp",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},
	"avif": {
		Extension: "avif",
		Mime:      "image/avif",
		Encoders: map[string]*media.Encoder{
			"jpg":  &Jpeg,
			"jpeg": &Jpeg,
			"webp": &Webp,
			"png":  &Png,
			"gif":  &Gif,
			"avif": &Avif,
		},
	},

	// Audio formats
	"mp3": {
		Extension: "mp3",
		Mime:      "audio/mpeg",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},
	"wav": {
		Extension: "wav",
		Mime:      "audio/wav",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},
	"flac": {
		Extension: "flac",
		Mime:      "audio/flac",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},
	"aac": {
		Extension: "aac",
		Mime:      "audio/aac",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},
	"ogg": {
		Extension: "ogg",
		Mime:      "audio/ogg",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},
	"m4a": {
		Extension: "m4a",
		Mime:      "audio/mp4",
		Encoders: map[string]*media.Encoder{
			"mp3":  &Mp3,
			"wav":  &Wav,
			"flac": &Flac,
			"aac":  &Aac,
			"ogg":  &Ogg,
			"m4a":  &M4a,
			"wma":  &Wma,
			"opus": &Opus,
			"json": &Json, // For metadata
		},
	},

	// Video formats
	"mp4": {
		Extension: "mp4",
		Mime:      "video/mp4",
		Encoders: map[string]*media.Encoder{
			"mp4":  &Mp4,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"webm": {
		Extension: "webm",
		Mime:      "video/webm",
		Encoders: map[string]*media.Encoder{
			"webm": &Webm,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"avi": {
		Extension: "avi",
		Mime:      "video/x-msvideo",
		Encoders: map[string]*media.Encoder{
			"avi":  &Avi,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"mov": {
		Extension: "mov",
		Mime:      "video/quicktime",
		Encoders: map[string]*media.Encoder{
			"mov":  &Mov,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"mkv": {
		Extension: "mkv",
		Mime:      "video/x-matroska",
		Encoders: map[string]*media.Encoder{
			"mkv":  &Mkv,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"flv": {
		Extension: "flv",
		Mime:      "video/x-flv",
		Encoders: map[string]*media.Encoder{
			"flv":  &Flv,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"wmv": {
		Extension: "wmv",
		Mime:      "video/x-ms-wmv",
		Encoders: map[string]*media.Encoder{
			"wmv":  &Wmv,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"m4v": {
		Extension: "m4v",
		Mime:      "video/x-m4v",
		Encoders: map[string]*media.Encoder{
			"m4v":  &M4v,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"3gp": {
		Extension: "3gp",
		Mime:      "video/3gpp",
		Encoders: map[string]*media.Encoder{
			"3gp": &ThreeGp,
			"jpg": &Jpeg, // For thumbnails
			"png": &Png,  // For thumbnails
		},
	},
	"ogv": {
		Extension: "ogv",
		Mime:      "video/ogg",
		Encoders: map[string]*media.Encoder{
			"ogv": &Ogv,
			"jpg": &Jpeg, // For thumbnails
			"png": &Png,  // For thumbnails
		},
	},

	// Document formats
	"pdf": {
		Extension: "pdf",
		Mime:      "application/pdf",
		Encoders: map[string]*media.Encoder{
			"pdf":  &Pdf,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},

	// Microsoft Office formats
	"docx": {
		Extension: "docx",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		Encoders: map[string]*media.Encoder{
			"docx": &Docx,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"xlsx": {
		Extension: "xlsx",
		Mime:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Encoders: map[string]*media.Encoder{
			"xlsx": &Xlsx,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"pptx": {
		Extension: "pptx",
		Mime:      "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		Encoders: map[string]*media.Encoder{
			"pptx": &Pptx,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},

	// Legacy Microsoft Office formats
	"doc": {
		Extension: "doc",
		Mime:      "application/msword",
		Encoders: map[string]*media.Encoder{
			"doc":  &Doc,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"xls": {
		Extension: "xls",
		Mime:      "application/vnd.ms-excel",
		Encoders: map[string]*media.Encoder{
			"xls":  &Xls,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"ppt": {
		Extension: "ppt",
		Mime:      "application/vnd.ms-powerpoint",
		Encoders: map[string]*media.Encoder{
			"ppt":  &Ppt,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},

	// OpenDocument formats
	"odt": {
		Extension: "odt",
		Mime:      "application/vnd.oasis.opendocument.text",
		Encoders: map[string]*media.Encoder{
			"odt":  &Odt,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"ods": {
		Extension: "ods",
		Mime:      "application/vnd.oasis.opendocument.spreadsheet",
		Encoders: map[string]*media.Encoder{
			"ods":  &Ods,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"odp": {
		Extension: "odp",
		Mime:      "application/vnd.oasis.opendocument.presentation",
		Encoders: map[string]*media.Encoder{
			"odp":  &Odp,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},

	// Text formats
	"txt": {
		Extension: "txt",
		Mime:      "text/plain",
		Encoders: map[string]*media.Encoder{
			"txt":  &Txt,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"rtf": {
		Extension: "rtf",
		Mime:      "application/rtf",
		Encoders: map[string]*media.Encoder{
			"rtf":  &Rtf,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"csv": {
		Extension: "csv",
		Mime:      "text/csv",
		Encoders: map[string]*media.Encoder{
			"csv":  &Csv,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},

	// Other common formats
	"epub": {
		Extension: "epub",
		Mime:      "application/epub+zip",
		Encoders: map[string]*media.Encoder{
			"epub": &Epub,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
	"xml": {
		Extension: "xml",
		Mime:      "application/xml",
		Encoders: map[string]*media.Encoder{
			"xml":  &Xml,
			"jpg":  &Jpeg, // For thumbnails
			"png":  &Png,  // For thumbnails
			"webp": &Webp, // For thumbnails
			"json": &Json, // For metadata
		},
	},
}
