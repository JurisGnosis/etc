package forward

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"e.coding.net/Love54dj/weizhong/etc/ryconn"
)

const TEST_SESSION_ID = "e5b21a57889541ffa01c6e387da971cd"
const VALID_REFERER = "https://servicewechat.com/"

func proxyHandlerMessageList(w http.ResponseWriter, r *http.Request) {
	// Use the client's request URL directly
	targetUrl := "http://47.107.101.100:9303/system/message/list"
	fmt.Println(r.URL.RawQuery)
	str, _ := json.Marshal(r)
	fmt.Println(str)
	if r.URL.RawQuery != "" {
		targetUrl += "?" + r.URL.RawQuery
	}

	// Check for invalid URL suffix
	if strings.HasSuffix(targetUrl, "senderId=") {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Check auth
	auth := r.Header.Get("Authorization")
	if !strings.HasSuffix(targetUrl, TEST_SESSION_ID) {
		if auth == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_, err := ryconn.AuthToMobile(auth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}

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

func proxyHandlerLogout(w http.ResponseWriter, r *http.Request) {
	targetUrl := "http://47.107.101.100:9303/logout"

	// Check auth
	referer := r.Header.Get("Referer")
	if !strings.HasPrefix(referer, VALID_REFERER) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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
	http.HandleFunc("/system/message/list", proxyHandlerMessageList)
	http.HandleFunc("/logout", proxyHandlerLogout)
	log.Printf("Starting proxy server on port :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
