package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// ExtractImagesFromPDFBytes extracts all embedded image streams from PDF bytes in pure Go.
func ExtractImagesFromPDFBytes(data []byte, maxPages int) ([][]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty pdf bytes")
	}

	r, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open pdf reader: %w", err)
	}

	num := r.NumPage()
	if maxPages > 0 && num > maxPages {
		num = maxPages
	}

	var images [][]byte
	for i := 1; i <= num; i++ {
		page := r.Page(i)
		xobjs := page.V.Key("Resources").Key("XObject")
		for _, k := range xobjs.Keys() {
			obj := xobjs.Key(k)
			if obj.Key("Subtype").Name() != "Image" {
				continue
			}

			imgBytes, err := readImageObject(obj)
			if err == nil && len(imgBytes) > 0 {
				images = append(images, imgBytes)
			}
		}
	}

	return images, nil
}

func readImageObject(obj Value) ([]byte, error) {
	rd := obj.Reader()
	data, err := io.ReadAll(rd)
	_ = rd.Close()
	if err != nil {
		return nil, err
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("stream too short")
	}

	if (data[0] == 0xFF && data[1] == 0xD8) || (data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G') {
		return data, nil
	}

	width := int(obj.Key("Width").Int64())
	height := int(obj.Key("Height").Int64())
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	if len(data) >= width*height*3 {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				idx := (y*width + x) * 3
				img.SetRGBA(x, y, color.RGBA{R: data[idx], G: data[idx+1], B: data[idx+2], A: 255})
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err == nil {
			return buf.Bytes(), nil
		}
	} else if len(data) >= width*height {
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				idx := y*width + x
				img.SetGray(x, y, color.Gray{Y: data[idx]})
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err == nil {
			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("unrecognized image encoding")
}
