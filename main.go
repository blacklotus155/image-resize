package main

import (
	"fmt"

	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/h2non/bimg"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file: " + err.Error())
	}

	cleanupInterval := time.Hour * 24 * 7 // One week
	directory := os.Getenv("BUCKET_STAGING_NAME")
	directory2 := os.Getenv("BUCKET_PRODUCTION_NAME")
	// Start a background process to delete the directory every week
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			err := removeAll(directory)
			if err != nil {
				fmt.Println("Error:", err)
			}
			fmt.Println("Directory", directory, "and its contents have been deleted.")

			err = removeAll(directory2)
			if err != nil {
				fmt.Println("Error:", err)
			}
			fmt.Println("Directory", directory2, "and its contents have been deleted.")
		}
	}()

	r := mux.NewRouter().SkipClean(true)
	r.HandleFunc("/favicon.ico", faviconHandler)
	r.PathPrefix("/").HandlerFunc(resizeHandler)
	http.Handle("/", r)
	fmt.Println("Server started on port 8080")
	http.ListenAndServe(":8080", r)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./favicon.ico")
}

func setImageSize(imgWidth, imgHeight int, widthStr, heightStr string) (width, height int) {
	defaultWidth := imgWidth
	defaultHeight := imgHeight
	var err error

	if widthStr != "" && heightStr == "" {
		defaultWidth, err = strconv.Atoi(widthStr)
		if err != nil || defaultWidth <= 0 {
			defaultWidth = imgWidth
		}
		defaultHeight = (defaultWidth / imgWidth) * imgHeight
	}
	if heightStr != "" {
		defaultHeight, err = strconv.Atoi(heightStr)
		if err != nil || defaultHeight <= 0 {
			defaultHeight = imgHeight
		}
		defaultWidth = (defaultHeight / imgHeight) * imgWidth
	}
	if heightStr != "" && widthStr != "" {
		defaultHeight, err = strconv.Atoi(heightStr)
		if err != nil || defaultHeight <= 0 {
			defaultHeight = imgHeight
		}
		defaultWidth, err = strconv.Atoi(widthStr)
		if err != nil || defaultWidth <= 0 {
			defaultWidth = imgWidth
		}
	}

	return defaultWidth, defaultHeight
}

func resizeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse provided width and height from query parameters
	widthStr := r.URL.Query().Get("width")
	heightStr := r.URL.Query().Get("height")
	formatStr := r.URL.Query().Get("format")
	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")
	isNeedWatermark := r.URL.Query().Get("watermark")

	// Extract image path from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "Image path is required", http.StatusBadRequest)
		return
	}
	objectKey := strings.Join(parts[2:], "/")
	bucket := parts[1]

	if strings.ToLower(bucket) == "staging" {
		bucket = os.Getenv("BUCKET_STAGING_NAME")
	} else if strings.ToLower(bucket) == "production" {
		bucket = os.Getenv("BUCKET_PRODUCTION_NAME")
	}
	baseURL := os.Getenv("BASE_URL")
	filePath := bucket + "/" + objectKey
	imageUrl := baseURL + filePath

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Fetch the image from URL
		resp, err := http.Get(imageUrl)
		if err != nil {
			http.Error(w, "Error fetching image "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Check if the response status code indicates success
		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to retrieve the image. ", http.StatusNotFound)
			return
		}

		directory := filepath.Dir(filePath)
		if err := os.MkdirAll(directory, 0755); err != nil {
			http.Error(w, "Error creating output directory:"+err.Error(), http.StatusInternalServerError)
			return
		}

		// Create the output file
		outputFile, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Error creating the output file:"+err.Error(), http.StatusInternalServerError)
			return
		}
		defer outputFile.Close()

		// Copy the response body to the output file
		_, err = io.Copy(outputFile, resp.Body)
		if err != nil {
			http.Error(w, "Error copying image data to file:"+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Read image file
	buffer, err := bimg.Read(filePath)
	if err != nil {
		http.Error(w, "Error read image", http.StatusInternalServerError)
		return
	}

	metadata, err := bimg.NewImage(buffer).Metadata()
	if err != nil {
		http.Error(w, "Error read metadata image", http.StatusInternalServerError)
		return
	}
	// Determine image dimensions
	imgWidth := metadata.Size.Width
	imgHeight := metadata.Size.Height
	imgFormat := metadata.Type

	// add watermarks to image
	if strings.Contains(objectKey, "uploads/charge_submission") || isNeedWatermark == "true" {
		if imgFormat != "jpeg" {
			buffer, err = bimg.NewImage(buffer).Convert(bimg.JPEG)
			if err != nil {
				http.Error(w, "Error converting image"+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		watermark := bimg.Watermark{
			Text:       "Baleomol.com",
			Opacity:    0.8,
			Width:      120,
			DPI:        150,
			Margin:     100,
			Font:       "sans bold 12",
			Background: bimg.Color{255, 255, 255},
		}
		buffer, err = bimg.NewImage(buffer).Watermark(watermark)
		if err != nil {
			http.Error(w, "Error adding watermark"+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Set default values based on image size
	defaultWidth := imgWidth
	defaultHeight := imgHeight

	// Parse provided width and height
	defaultWidth, defaultHeight = setImageSize(imgWidth, imgHeight, widthStr, heightStr)
	defaultWidth, defaultHeight = setImageSize(defaultWidth, defaultHeight, wStr, hStr)

	if formatStr != "" {
		imgFormat = formatStr
	}

	var extension bimg.ImageType
	switch imgFormat {
	case "png":
		extension = bimg.PNG
	case "webp":
		extension = bimg.WEBP
	default:
		extension = bimg.JPEG
	}

	if !bimg.IsTypeNameSupported(imgFormat) {
		http.Error(w, "Error image type unsupported", http.StatusInternalServerError)
		return
	}

	contentType := "image/" + bimg.ImageTypeName(extension)
	// process options
	options := bimg.Options{
		Width:       defaultWidth,
		Height:      defaultHeight,
		Compression: 8,
		Type:        extension,
	}
	// Encode the resized image
	buffer, err = bimg.NewImage(buffer).Process(options)
	if err != nil {
		http.Error(w, "Error processing image"+err.Error(), http.StatusInternalServerError)
		return
	}
	// Set content type header
	w.Header().Set("Content-Type", contentType)
	_, err = w.Write(buffer)

	if err != nil {
		http.Error(w, "Error serve image", http.StatusInternalServerError)
		return
	}
}

func removeAll(directory string) error {
	err := os.RemoveAll(directory)
	return err
}
