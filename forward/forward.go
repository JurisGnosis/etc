package forward

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetUrl := r.URL.Query().Get("url")

	// Validate URL is not empty
	fmt.Println("targetUrl:", targetUrl)
	if targetUrl == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	// Check for invalid URL suffix
	if strings.HasSuffix(targetUrl, "senderId=") {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Ensure URL has a protocol
	if !strings.HasPrefix(targetUrl, "http://") && !strings.HasPrefix(targetUrl, "https://") {
		targetUrl = "http://" + targetUrl
	}

	// URL replacements for redirecting to the correct backend
	targetUrl = strings.Replace(targetUrl, "lawyer.dlaws.cn:9900/", "47.107.101.100:9303/", 1)
	targetUrl = strings.Replace(targetUrl, "47.107.101.100:9304/", "47.107.101.100:9303/", 1)
	fmt.Println("targetUrl:", targetUrl)

	// Create new request
	req, err := http.NewRequest(r.Method, targetUrl, r.Body)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers from client request
	for key, value := range r.Header {
		req.Header[key] = value
	}

	// Send request to target server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request to %s: %v", targetUrl, err)
		http.Error(w, "Error making request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		http.Error(w, "Error reading response body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers and status code
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Write response body to client
	_, err = w.Write(body)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func ServeOnPort(port int) {
	http.HandleFunc("/", proxyHandler)
	log.Printf("Starting proxy server on port :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
