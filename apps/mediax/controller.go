package mediax

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/outcome"
	"github.com/getevo/evo/v2/lib/text"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"mediax/apps/media"
	"path/filepath"
	"strings"
)

type Controller struct{}

func (c Controller) ServeMedia(request *evo.Request) any {
	// Block until the first InitializeConfig completes. After that, the channel
	// is permanently closed so this is a no-op for every subsequent request.
	<-ready

	var url = request.URL()
	var req media.Request

	// Generate trace ID for this request
	traceID := uuid.New().String()
	request.Set("X-Trace-ID", traceID)

	// Check if debugging is enabled
	debugEnabled := request.Header("X-Debug") == "1"

	if debugEnabled {
		log.Debug("Request started", "trace_id", traceID, "host", url.Host, "path", url.Path)
		request.Set("X-Debug-Host", url.Host)
	}

	if v, ok := lookupOrigin(url.Host); ok {
		req = media.Request{
			Request:   request,
			Domain:    url.Host,
			Url:       url,
			Origin:    v,
			Extension: strings.ToLower(filepath.Ext(url.Path)),
			Debug:     debugEnabled,
			TraceID:   traceID,
		}
		if len(req.Origin.Storages) == 0 {
			return outcome.Text("no storages configured for this domain").Status(evo.StatusInternalServerError)
		}
		extension, err := GetURLExtension(req.Url.Path)
		if req.Debug {
			log.Debug("URL extension parsed", "trace_id", traceID, "extension", extension)
			request.Set("X-Debug-Extension", extension)
		}
		if err != nil {
			if req.Debug {
				log.Debug("Unsupported media type", "trace_id", traceID, "error", err.Error())
				request.Set("X-Debug-Error", "unsupported media type: "+err.Error())
			}
			return outcome.Text("unsupported media type").Status(evo.StatusUnsupportedMediaType)
		}
		req.Extension = extension
	} else {
		return outcome.Text("forbidden domain").Status(evo.StatusForbidden)
	}

	var ok bool
	if req.MediaType, ok = MediaTypes[req.Extension]; !ok {
		return outcome.Text("unsupported media type").Status(415)
	}

	options, err := req.MediaType.ParseOptions(request)
	if err != nil {
		return err
	}
	req.Options = options
	if req.Debug {
		log.Debug("Media processing details", "trace_id", traceID, "media_type", text.ToJSON(req.MediaType), "options", text.ToJSON(req.Options))
		request.Set("X-Debug-MediaType", text.ToJSON(req.MediaType))
		request.Set("X-Debug-Options", text.ToJSON(req.Options))
	}
	req.OriginalFilePath = TrimPrefix(req.Url.Path, req.Origin.PrefixPath)

	//stage the file
	err = req.StageFile()
	if err != nil {
		if req.StagedFilePath == media.STAGING {
			req.Request.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			req.Request.Set("Expires", "0")
			req.Request.Set("Pragma", "no-cache")
			req.Request.Set("Location", req.Url.Path+"?"+req.Request.QueryString())
			req.Request.Status(evo.StatusTemporaryRedirect)
			return outcome.Response{}
		}
		req.Request.Status(evo.StatusNotFound)
		return fmt.Errorf("file not found")
	}
	var encoder = options.Encoder
	if encoder.Processor != nil {

		err = encoder.Processor(&req)
		if err != nil {
			return err
		}

		// Check if detail=true and we have metadata to return
		if options.Detail && len(req.Metadata) > 0 {
			// Return metadata as JSON
			request.Set("Content-Type", "application/json")
			request.Status(fiber.StatusOK)
			return req.Metadata
		}

		// Use ProcessedMimeType if available (e.g., for thumbnails), otherwise use encoder's MIME type
		mimeType := encoder.Mime
		if req.ProcessedMimeType != "" {
			mimeType = req.ProcessedMimeType
		}

		err = req.ServeFile(mimeType, req.ProcessedFilePath)
		if err != nil {
			return err
		}

	} else {
		err = req.ServeFile(encoder.Mime, req.StagedFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func TrimPrefix(url, prefix string) string {
	if len(url) >= len(prefix) && url[:len(prefix)] == prefix {
		return strings.Trim(url[len(prefix):], `\/`)
	}
	return url
}
