package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var (
	mu    sync.Mutex
	cache = make(map[string]string)
)

func downloadFile(url, filePath string, wg *sync.WaitGroup) error {
	defer wg.Done()

	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects
			return nil
		},
	}

	response, err := client.Get(url)
	if err != nil {
		log.Printf("Error downloading file %s: %v", url, err)
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Printf("Error copying content to file %s: %v", filePath, err)
		return err
	}

	log.Printf("Downloaded %s to %s", url, filePath)
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	decodedURL, err := url.QueryUnescape(urlParam)
	if err != nil {
		http.Error(w, "Error decoding 'url' parameter", http.StatusBadRequest)
		return
	}

	mu.Lock()
	filePath, exists := cache[decodedURL]
	mu.Unlock()

	if !exists {
		// Create the 'temp' directory if it doesn't exist
		if err := os.MkdirAll("temp", os.ModePerm); err != nil {
			http.Error(w, fmt.Sprintf("Error creating 'temp' directory: %v", err), http.StatusInternalServerError)
			return
		}

		filePath = filepath.Join("temp", filepath.Base(decodedURL))
		var wg sync.WaitGroup
		wg.Add(1)

		err := downloadFile(decodedURL, filePath, &wg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error downloading file: %v", err), http.StatusInternalServerError)
			return
		}

		wg.Wait()

		mu.Lock()
		cache[decodedURL] = filePath
		mu.Unlock()
	}

	http.ServeFile(w, r, filePath)
}

func wipeCacheHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	for _, filePath := range cache {
		err := os.Remove(filePath)
		if err != nil {
			log.Printf("Error removing file %s: %v", filePath, err)
		}
	}

	cache = make(map[string]string)

	log.Println("Cache wiped successfully.")
	fmt.Fprint(w, "Cache wiped successfully.")
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.HandleFunc("/wipe-cache", wipeCacheHandler)

	// Create a CORS handler with default options
	c := cors.Default()

	// Attach the CORS handler to the router
	handlerWithCors := c.Handler(r)

	port := 8080
	log.Printf("Starting download proxy on :%d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), handlerWithCors)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
