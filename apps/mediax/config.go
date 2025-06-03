package mediax

import (
	"mediax/apps/media"
	"mediax/encoders"
)

var MediaTypes = map[string]*media.Type{
	"jpg": {
		Extension: "jpg",
		Mime:      "image/jpeg",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp},
	},
	"png": {
		Extension: "png",
		Mime:      "image/png",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp},
	},
	"gif": {
		Extension: "gif",
		Mime:      "image/gif",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp},
	},
	"webp": {
		Extension: "webp",
		Mime:      "image/webp",
		Encoders:  map[string]*media.Encoder{"jpg": &encoders.Jpeg, "png": &encoders.Png, "gif": &encoders.Gif, "webp": &encoders.Webp},
	},
}
