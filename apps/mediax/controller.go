package mediax

import (
	"fmt"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/outcome"
	"github.com/getevo/evo/v2/lib/text"
	"mediax/apps/media"
	"path/filepath"
	"strings"
)

type Controller struct{}

func (c Controller) ServeMedia(request *evo.Request) any {
	Wait.Wait()

	var url = request.URL()
	var req media.Request

	if v, ok := Origins[url.Host]; ok {
		req = media.Request{
			Request:   request,
			Domain:    url.Host,
			Url:       url,
			Origin:    v,
			Extension: strings.ToLower(filepath.Ext(url.Path)),
			Debug:     request.Header("X-Debug") == "1",
		}
		if len(req.Origin.Storages) == 0 {
			return outcome.Text("no storages configured for this domain").Status(evo.StatusInternalServerError)
		}
		extension, err := GetURLExtension(req.Url.Path)
		if req.Debug {
			fmt.Println("-------------------")
			fmt.Println("Extension:", extension)
		}
		if err != nil {
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
		fmt.Println("MediaType:", text.ToJSON(req.MediaType))
		fmt.Println("Options:", text.ToJSON(req.Options))
	}
	req.OriginalFilePath = TrimPrefix(req.Url.Path, req.Origin.PrefixPath)

	//stage the file
	err = req.StageFile()
	if err != nil {
		req.Request.Status(evo.StatusNotFound)
		return fmt.Errorf("file not found")
	}

	var encoder = options.Encoder
	if encoder.Processor != nil {

		err = encoder.Processor(&req)
		if err != nil {
			return err
		}

		err = req.ServeFile(encoder.Mime, req.ProcessedFilePath)
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
