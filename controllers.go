package main

import (
	"encoding/json"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/h2non/bimg"
	"github.com/h2non/filetype"
)

func indexController(o ServerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path.Join(o.PathPrefix, "/") {
			ErrorReply(r, w, ErrNotFound, ServerOptions{})
			return
		}

		body, _ := json.Marshal(Versions{
			Version,
			bimg.Version,
			bimg.VipsVersion,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func healthController(w http.ResponseWriter, r *http.Request) {
	health := GetHealthStats()
	body, _ := json.Marshal(health)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func imageController(o ServerOptions, operation Operation) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		imageSource := MatchSource(req)
		if imageSource == nil {
			ErrorReply(req, w, ErrMissingImageSource, o)
			return
		}

		buf, err := imageSource.GetImage(req)
		if err != nil {
			if xerr, ok := err.(Error); ok {
				ErrorReply(req, w, xerr, o)
			} else {
				ErrorReply(req, w, NewError(err.Error(), http.StatusBadRequest), o)
			}
			return
		}

		if len(buf) == 0 {
			ErrorReply(req, w, ErrEmptyBody, o)
			return
		}

		imageHandler(w, req, buf, operation, o)
	}
}

func determineAcceptMimeType(accept string) string {
	for _, v := range strings.Split(accept, ",") {
		mediaType, _, _ := mime.ParseMediaType(v)
		switch mediaType {
		case "image/webp":
			return "webp"
		case "image/png":
			return "png"
		case "image/jpeg":
			return "jpeg"
		case "image/avif":
			return "avif"
		}
	}

	return ""
}

func imageHandler(w http.ResponseWriter, r *http.Request, buf []byte, operation Operation, o ServerOptions) {
	// Infer the body MIME type via mime sniff algorithm
	mimeType := http.DetectContentType(buf)

	// If cannot infer the type, infer it via magic numbers
	if mimeType == "application/octet-stream" {
		kind, err := filetype.Get(buf)
		if err == nil && kind.MIME.Value != "" {
			mimeType = kind.MIME.Value
		}
	}

	// Infer text/plain responses as potential SVG image
	if strings.Contains(mimeType, "text/plain") && len(buf) > 8 {
		if bimg.IsSVGImage(buf) {
			mimeType = "image/svg+xml"
		}
	}

	// Finally check if image MIME type is supported
	if !IsImageMimeTypeSupported(mimeType) {
		ErrorReply(r, w, ErrUnsupportedMedia, o)
		return
	}

	opts, err := buildParamsFromQuery(r.URL.Query())
	if err != nil {
		ErrorReply(r, w, NewError("Error while processing parameters, "+err.Error(), http.StatusBadRequest), o)
		return
	}

	vary := ""
	if opts.Type == "auto" {
		opts.Type = determineAcceptMimeType(r.Header.Get("Accept"))
		vary = "Accept" // Ensure caches behave correctly for negotiated content
	} else if opts.Type != "" && ImageType(opts.Type) == 0 {
		ErrorReply(r, w, ErrOutputFormat, o)
		return
	}

	image, err := operation.Run(buf, opts)
	if err != nil {
		// Ensure the Vary header is set when an error occurs
		if vary != "" {
			w.Header().Set("Vary", vary)
		}
		ErrorReply(r, w, NewError("Error while processing the image: "+err.Error(), http.StatusBadRequest), o)
		return
	}

	// Expose Content-Length response header
	w.Header().Set("Content-Length", strconv.Itoa(len(image.Body)))
	w.Header().Set("Content-Type", image.Mime)
	if image.Mime != "application/json" && o.ReturnSize {
		meta, err := bimg.Metadata(image.Body)
		if err == nil {
			w.Header().Set("Image-Width", strconv.Itoa(meta.Size.Width))
			w.Header().Set("Image-Height", strconv.Itoa(meta.Size.Height))
		}
	}
	if vary != "" {
		w.Header().Set("Vary", vary)
	}
	_, _ = w.Write(image.Body)
}
