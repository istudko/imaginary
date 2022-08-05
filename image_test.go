package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestImageResize(t *testing.T) {
	t.Run("Width and Height defined", func(t *testing.T) {
		opts := ImageOptions{Width: 300, Height: 300}
		buf, _ := io.ReadAll(readFile("imaginary.jpg"))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
			return
		}
		if img.Mime != "image/jpeg" {
			t.Error("Invalid image MIME type")
		}
		if assertSize(img.Body, opts.Width, opts.Height) != nil {
			t.Errorf("Invalid image size, expected: %dx%d", opts.Width, opts.Height)
		}
	})

	t.Run("Width defined", func(t *testing.T) {
		opts := ImageOptions{Width: 300}
		buf, _ := io.ReadAll(readFile("imaginary.jpg"))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
			return
		}
		if img.Mime != "image/jpeg" {
			t.Error("Invalid image MIME type")
		}
		if err := assertSize(img.Body, 300, 404); err != nil {
			t.Error(err)
		}
	})

	t.Run("Width defined with NoCrop=false", func(t *testing.T) {
		opts := ImageOptions{Width: 300, NoCrop: false, IsDefinedField: IsDefinedField{NoCrop: true}}
		buf, _ := io.ReadAll(readFile("imaginary.jpg"))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
			return
		}
		if img.Mime != "image/jpeg" {
			t.Error("Invalid image MIME type")
		}

		// The original image is 550x740
		if err := assertSize(img.Body, 300, 740); err != nil {
			t.Error(err)
		}
	})

	t.Run("Width defined with NoCrop=true", func(t *testing.T) {
		opts := ImageOptions{Width: 300, NoCrop: true, IsDefinedField: IsDefinedField{NoCrop: true}}
		buf, _ := io.ReadAll(readFile("imaginary.jpg"))

		img, err := Resize(buf, opts)
		if err != nil {
			t.Errorf("Cannot process image: %s", err)
			return
		}
		if img.Mime != "image/jpeg" {
			t.Error("Invalid image MIME type")
		}

		// The original image is 550x740
		if err := assertSize(img.Body, 300, 404); err != nil {
			t.Error(err)
		}
	})

}

func TestImageFit(t *testing.T) {
	opts := ImageOptions{Width: 300, Height: 300}
	buf, _ := io.ReadAll(readFile("imaginary.jpg"))

	img, err := Fit(buf, opts)
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
		return
	}
	if img.Mime != "image/jpeg" {
		t.Error("Invalid image MIME type")
	}
	// 550x740 -> 222.9x300
	if assertSize(img.Body, 223, 300) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", opts.Width, opts.Height)
	}
}

func TestImageAutoRotate(t *testing.T) {
	buf, _ := io.ReadAll(readFile("imaginary.jpg"))
	img, err := AutoRotate(buf, ImageOptions{})
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
		return
	}
	if img.Mime != "image/jpeg" {
		t.Error("Invalid image MIME type")
	}
	if assertSize(img.Body, 550, 740) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", 550, 740)
	}
}

func TestImagePipelineOperations(t *testing.T) {
	width, height := 300, 260

	operations := PipelineOperations{
		PipelineOperation{
			Name: "crop",
			Params: map[string]interface{}{
				"width":  width,
				"height": height,
			},
		},
		PipelineOperation{
			Name: "convert",
			Params: map[string]interface{}{
				"type": "webp",
			},
		},
	}

	opts := ImageOptions{Operations: operations}
	buf, _ := io.ReadAll(readFile("imaginary.jpg"))

	img, err := Pipeline(buf, opts)
	if err != nil {
		t.Errorf("Cannot process image: %s", err)
		return
	}
	if img.Mime != "image/webp" {
		t.Error("Invalid image MIME type")
	}
	if assertSize(img.Body, width, height) != nil {
		t.Errorf("Invalid image size, expected: %dx%d", width, height)
	}
}

func TestImageMultiTasks(t *testing.T) {
	tasks := []MultiTask{
		{
			Name:          "info",
			OperationName: "info",
		},
		{
			Name:          "make-smaller",
			OperationName: "resize",
			Params: map[string]interface{}{
				"width": 300,
				"type":  "webp",
			},
		},
	}

	opts := ImageOptions{Multi: tasks}
	buf, _ := io.ReadAll(readFile("imaginary.jpg"))

	mp, err := Multi(buf, opts)
	if err != nil {
		t.Errorf("Cannot process tasks: %s", err)
		return
	}

	mimeType, mimeParams, err := mime.ParseMediaType(mp.Mime)
	if err != nil {
		t.Errorf("Cannot parse mime type: %s", err)
		return
	}
	if mimeType != "multipart/form-data" || mimeParams["boundary"] == "" {
		t.Error("Invalid MIME type")
		return
	}

	mr := multipart.NewReader(bytes.NewReader(mp.Body), mimeParams["boundary"])
	var found int
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error getting next part: %s", err)
			return
		}

		data, err := io.ReadAll(p)
		if err != nil {
			t.Errorf("Error reading multipart data: %s", err)
			return
		}

		switch p.FormName() {
		case "info":
			found++
			if p.Header.Get("content-type") != "application/json" {
				t.Error("Part's content type is not application/json", p.Header.Get("content-type"))
				return
			}
			var imageInfo ImageInfo
			err = json.Unmarshal(data, &imageInfo)
			if err != nil {
				t.Errorf("Error parsing image info: %s", err)
				return
			}
			if imageInfo.Width != 550 || imageInfo.Height != 740 || imageInfo.EXIF.Orientation != 1 {
				t.Error("Unexpected metadata values", string(data))
				return
			}
		case "make-smaller":
			found++
			if !strings.HasSuffix(p.FileName(), ".webp") {
				t.Error("Part's file name does not end in .webp", p.FileName())
				return
			}
			if p.Header.Get("content-type") != "image/webp" {
				t.Error("Part's content type is not image/webp", p.Header.Get("content-type"))
				return
			}
			if err = assertSize(data, 300, 404); err != nil {
				t.Error(err)
				return
			}
		default:
			t.Error("Found foreign part: " + p.FormName())
			return
		}
	}

	if found != 2 {
		t.Error("Expected to find 2 parts, but found", found)
		return
	}
}

func TestCalculateDestinationFitDimension(t *testing.T) {
	cases := []struct {
		// Image
		imageWidth  int
		imageHeight int

		// User parameter
		optionWidth  int
		optionHeight int

		// Expect
		fitWidth  int
		fitHeight int
	}{

		// Leading Width
		{1280, 1000, 710, 9999, 710, 555},
		{1279, 1000, 710, 9999, 710, 555},
		{900, 500, 312, 312, 312, 173}, // rounding down
		{900, 500, 313, 313, 313, 174}, // rounding up

		// Leading height
		{1299, 2000, 710, 999, 649, 999},
		{1500, 2000, 710, 999, 710, 947},
	}

	for _, tc := range cases {
		fitWidth, fitHeight := calculateDestinationFitDimension(tc.imageWidth, tc.imageHeight, tc.optionWidth, tc.optionHeight)
		if fitWidth != tc.fitWidth || fitHeight != tc.fitHeight {
			t.Errorf(
				"Fit dimensions calculation failure\nExpected : %d/%d (width/height)\nActual   : %d/%d (width/height)\n%+v",
				tc.fitWidth, tc.fitHeight, fitWidth, fitHeight, tc,
			)
		}
	}

}
