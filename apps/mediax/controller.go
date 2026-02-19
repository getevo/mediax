package mediax

import (
	"bytes"
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/outcome"
	"github.com/getevo/evo/v2/lib/text"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"mediax/apps/media"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Controller struct{}

func (c Controller) ServeMedia(request *evo.Request) any {
	var url = request.URL()

	// Fiber's /* wildcard catches all GET requests including specific routes.
	// Handle known non-media paths before blocking on ready.
	if url.Path == "/health" {
		return outcome.Json(map[string]string{"status": "ok"})
	}

	// Pass admin paths through to restify routes.
	if strings.HasPrefix(url.Path, "/admin") {
		return request.Next()
	}

	// Block until the first InitializeConfig completes. After that, the channel
	// is permanently closed so this is a no-op for every subsequent request.
	<-ready

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
		return outcome.Text("unsupported media type").Status(evo.StatusUnsupportedMediaType)
	}

	options, err := req.MediaType.ParseOptions(request)
	if err != nil {
		return err
	}
	if options.Profile != "" {
		if vp, ok := lookupVideoProfile(options.Profile); ok {
			options.VideoProfile = vp
		} else {
			return outcome.Text("unknown video profile: " + options.Profile).Status(evo.StatusBadRequest)
		}
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
		return fmt.Errorf("file not found: %w", err)
	}
	var encoder = options.Encoder
	if encoder.Processor != nil {
		procStart := time.Now()
		err = encoder.Processor(&req)
		metricProcessingDuration.WithLabelValues(req.Extension).Observe(time.Since(procStart).Seconds())
		if err != nil {
			metricRequests.WithLabelValues(req.Extension, "error").Inc()
			return err
		}

		// Check if detail=true and we have metadata to return
		if options.Detail && len(req.Metadata) > 0 {
			// Return metadata as JSON
			request.Set("Content-Type", "application/json")
			request.Status(fiber.StatusOK)
			metricRequests.WithLabelValues(req.Extension, "ok").Inc()
			return req.Metadata
		}

		// Use ProcessedMimeType if available (e.g., for thumbnails), otherwise use encoder's MIME type
		mimeType := encoder.Mime
		if req.ProcessedMimeType != "" {
			mimeType = req.ProcessedMimeType
		}

		// Resolve the file to serve: fall back to the staged file when the
		// processor returns nil without setting ProcessedFilePath (e.g. video
		// pass-through when no preview/thumbnail option was requested).
		serveFilePath := req.ProcessedFilePath
		if serveFilePath == "" {
			serveFilePath = req.StagedFilePath
		} else if _, statErr := os.Stat(serveFilePath); statErr != nil {
			metricRequests.WithLabelValues(req.Extension, "error").Inc()
			return fmt.Errorf("processor did not produce output file: %w", statErr)
		}

		err = req.ServeFile(mimeType, serveFilePath)
		if err != nil {
			metricRequests.WithLabelValues(req.Extension, "error").Inc()
			return err
		}

	} else {
		err = req.ServeFile(encoder.Mime, req.StagedFilePath)
		if err != nil {
			metricRequests.WithLabelValues(req.Extension, "error").Inc()
			return err
		}
	}
	metricRequests.WithLabelValues(req.Extension, "ok").Inc()
	return nil
}

// PrometheusMetrics serves Prometheus-format metrics at /prometheus/metrics.
func (c Controller) PrometheusMetrics(request *evo.Request) any {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil && len(mfs) == 0 {
		return err
	}
	var buf bytes.Buffer
	for _, mf := range mfs {
		if _, encErr := expfmt.MetricFamilyToText(&buf, mf); encErr != nil {
			break
		}
	}
	request.Context.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	request.Context.Status(fiber.StatusOK)
	request.Context.Write(buf.Bytes()) //nolint:errcheck
	return nil
}

func (c Controller) Health(request *evo.Request) any {
	return outcome.Json(map[string]string{"status": "ok"})
}

func (c Controller) Reload(request *evo.Request) any {
	go InitializeConfig()
	return outcome.Json(map[string]string{"status": "reloading"})
}

func TrimPrefix(url, prefix string) string {
	return strings.Trim(strings.TrimPrefix(url, prefix), `\/`)
}
