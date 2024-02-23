package main

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"net/http"
	"strconv"
	"strings"

	// "github.com/kolesa-team/go-webp/encoder"
	// "github.com/kolesa-team/go-webp/webp"
	"github.com/chai2010/webp"
	"github.com/nfnt/resize"
)

func main() {
	http.HandleFunc("/", resizeHandler)
	fmt.Println("Server started on port 8080")
	http.ListenAndServe(":8080", nil)
}

func resizeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract image path from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "Image path is required", http.StatusBadRequest)
		return
	}
	objectKey := strings.Join(parts[2:], "/")
	bucket := parts[1]

	baseURL := "https://images.baleomol.com/"
	imageUrl := baseURL + bucket + "/" + objectKey

	// Fetch the image from URL
	resp, err := http.Get(imageUrl)
	if err != nil {
		http.Error(w, "Error fetching image"+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Decode the image
	img, format, err := image.Decode(resp.Body)
	if err != nil {
		http.Error(w, "Error decoding image"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine image dimensions
	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()

	// Parse provided width and height from query parameters
	widthStr := r.URL.Query().Get("width")
	heightStr := r.URL.Query().Get("height")
	formatStr := r.URL.Query().Get("format")
	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")

	// Set default values based on image size
	defaultWidth := imgWidth
	defaultHeight := imgHeight

	// Parse provided width and height
	if widthStr != "" {
		defaultWidth, err = strconv.Atoi(widthStr)
		if err != nil || defaultWidth <= 0 {
			defaultWidth = imgWidth
		}
	}
	if heightStr != "" {
		defaultHeight, err = strconv.Atoi(heightStr)
		if err != nil || defaultHeight <= 0 {
			defaultHeight = imgHeight
		}
	}
	if wStr != "" {
		defaultWidth, err = strconv.Atoi(wStr)
		if err != nil || defaultWidth <= 0 {
			defaultWidth = imgWidth
		}
	}
	if hStr != "" {
		defaultHeight, err = strconv.Atoi(hStr)
		if err != nil || defaultHeight <= 0 {
			defaultHeight = imgHeight
		}
	}
	if formatStr != "" {
		format = formatStr
	}

	// Resize the image
	resizedImg := resize.Resize(uint(defaultWidth), uint(defaultHeight), img, resize.Lanczos3)

	// Determine content type
	var contentType string
	switch format {
	case "jpeg":
		contentType = "image/jpeg"
	case "png":
		contentType = "image/png"
	case "webp":
		contentType = "image/webp"
	default:
		contentType = "image/jpeg"
	}

	// Set content type header
	w.Header().Set("Content-Type", contentType)

	// Encode the resized image
	switch contentType {
	case "image/jpeg":
		err = jpeg.Encode(w, resizedImg, nil)
	case "image/png":
		err = png.Encode(w, resizedImg)
	case "image/webp":
		var data []byte
		data, err = webp.EncodeLosslessRGB(resizedImg)
		_, err = w.Write(data)

		// err = webp.encode(w, m, &webp.Options{Lossless: true})
		// options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
		// if err != nil {
		// 	http.Error(w, "Failed create options", http.StatusInternalServerError)
		// 	return
		// }
		// err = webp.Encode(w, resizedImg, options)
	default:
		err = jpeg.Encode(w, resizedImg, nil)
	}
	if err != nil {
		http.Error(w, "Error encoding image", http.StatusInternalServerError)
		return
	}
}
