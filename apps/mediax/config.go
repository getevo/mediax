package mediax

import (
	"mediax/apps/media"
	"mediax/encoders"
)

var MediaTypes = map[string]*media.Type{
	"jpg": {
		Extension: "jpg",
		Mime:      "image/jpeg",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp, "avif": &encoders.Avif},
	},
	"png": {
		Extension: "png",
		Mime:      "image/png",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp, "avif": &encoders.Avif},
	},
	"gif": {
		Extension: "gif",
		Mime:      "image/gif",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp, "avif": &encoders.Avif},
	},
	"webp": {
		Extension: "webp",
		Mime:      "image/webp",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp, "avif": &encoders.Avif},
	},
	"avif": {
		Extension: "avif",
		Mime:      "image/avif",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp, "avif": &encoders.Avif},
	},
	// Video formats
	"mp4": {
		Extension: "mp4",
		Mime:      "video/mp4",
		Encoders:  map[string]*media.Encoder{"mp4": &encoders.Mp4, "jpg": &encoders.Mp4, "png": &encoders.Mp4, "webp": &encoders.Mp4, "avif": &encoders.Mp4},
	},
	"webm": {
		Extension: "webm",
		Mime:      "video/webm",
		Encoders:  map[string]*media.Encoder{"webm": &encoders.Webm, "jpg": &encoders.Webm, "png": &encoders.Webm, "webp": &encoders.Webm, "avif": &encoders.Webm},
	},
	"avi": {
		Extension: "avi",
		Mime:      "video/x-msvideo",
		Encoders:  map[string]*media.Encoder{"avi": &encoders.Avi, "jpg": &encoders.Avi, "png": &encoders.Avi, "webp": &encoders.Avi, "avif": &encoders.Avi},
	},
	"mov": {
		Extension: "mov",
		Mime:      "video/quicktime",
		Encoders:  map[string]*media.Encoder{"mov": &encoders.Mov, "jpg": &encoders.Mov, "png": &encoders.Mov, "webp": &encoders.Mov, "avif": &encoders.Mov},
	},
	"mkv": {
		Extension: "mkv",
		Mime:      "video/x-matroska",
		Encoders:  map[string]*media.Encoder{"mkv": &encoders.Mkv, "jpg": &encoders.Mkv, "png": &encoders.Mkv, "webp": &encoders.Mkv, "avif": &encoders.Mkv},
	},
	"flv": {
		Extension: "flv",
		Mime:      "video/x-flv",
		Encoders:  map[string]*media.Encoder{"flv": &encoders.Flv, "jpg": &encoders.Flv, "png": &encoders.Flv, "webp": &encoders.Flv, "avif": &encoders.Flv},
	},
	"wmv": {
		Extension: "wmv",
		Mime:      "video/x-ms-wmv",
		Encoders:  map[string]*media.Encoder{"wmv": &encoders.Wmv, "jpg": &encoders.Wmv, "png": &encoders.Wmv, "webp": &encoders.Wmv, "avif": &encoders.Wmv},
	},
	"m4v": {
		Extension: "m4v",
		Mime:      "video/x-m4v",
		Encoders:  map[string]*media.Encoder{"m4v": &encoders.M4v, "jpg": &encoders.M4v, "png": &encoders.M4v, "webp": &encoders.M4v, "avif": &encoders.M4v},
	},
	"3gp": {
		Extension: "3gp",
		Mime:      "video/3gpp",
		Encoders:  map[string]*media.Encoder{"3gp": &encoders.ThreeGp, "jpg": &encoders.ThreeGp, "png": &encoders.ThreeGp, "webp": &encoders.ThreeGp, "avif": &encoders.ThreeGp},
	},
	"ogv": {
		Extension: "ogv",
		Mime:      "video/ogg",
		Encoders:  map[string]*media.Encoder{"ogv": &encoders.Ogv, "jpg": &encoders.Ogv, "png": &encoders.Ogv, "webp": &encoders.Ogv, "avif": &encoders.Ogv},
	},
	// Audio formats with conversion support
	"mp3": {
		Extension: "mp3",
		Mime:      "audio/mpeg",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Mp3, "png": &encoders.Mp3, "webp": &encoders.Mp3, "avif": &encoders.Mp3},
	},
	"wav": {
		Extension: "wav",
		Mime:      "audio/wav",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Wav, "png": &encoders.Wav, "webp": &encoders.Wav, "avif": &encoders.Wav},
	},
	"flac": {
		Extension: "flac",
		Mime:      "audio/flac",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Flac, "png": &encoders.Flac, "webp": &encoders.Flac, "avif": &encoders.Flac},
	},
	"aac": {
		Extension: "aac",
		Mime:      "audio/aac",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Aac, "png": &encoders.Aac, "webp": &encoders.Aac, "avif": &encoders.Aac},
	},
	"ogg": {
		Extension: "ogg",
		Mime:      "audio/ogg",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Ogg, "png": &encoders.Ogg, "webp": &encoders.Ogg, "avif": &encoders.Ogg},
	},
	"m4a": {
		Extension: "m4a",
		Mime:      "audio/mp4",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.M4a, "png": &encoders.M4a, "webp": &encoders.M4a, "avif": &encoders.M4a},
	},
	"wma": {
		Extension: "wma",
		Mime:      "audio/x-ms-wma",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Wma, "png": &encoders.Wma, "webp": &encoders.Wma, "avif": &encoders.Wma},
	},
	"opus": {
		Extension: "opus",
		Mime:      "audio/opus",
		Encoders:  map[string]*media.Encoder{"mp3": &encoders.Mp3, "wav": &encoders.Wav, "flac": &encoders.Flac, "aac": &encoders.Aac, "ogg": &encoders.Ogg, "m4a": &encoders.M4a, "wma": &encoders.Wma, "opus": &encoders.Opus, "jpg": &encoders.Opus, "png": &encoders.Opus, "webp": &encoders.Opus, "avif": &encoders.Opus},
	},
}
