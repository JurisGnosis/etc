package forward

import (
	"bytes"
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

// whatever
const loginInfoUrl = "http://47.107.101.100:9303/system/loginInfo"
const baseUrl = "http://47.107.101.100:9303"
const fixedLawyerId = "132"

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

	// Get userId
	userId, err := GetIdByAuth(auth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Check senderId
	requestSenderId := r.URL.Query().Get("senderId")
	standardSenderId, err := GetSenderIdByAuth(userId, auth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if requestSenderId != standardSenderId {
		http.Error(w, "Invalid senderId", http.StatusUnauthorized)
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

func GetIdByAuth(auth string) (string, error) {
	req, err := http.NewRequest("GET", loginInfoUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", auth)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Print the response body
	fmt.Println("Response body:", string(bodyBytes))

	// Create a new reader from the bytes for JSON decoding
	bodyReader := bytes.NewReader(bodyBytes)

	var data struct {
		Data struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bodyReader).Decode(&data); err != nil {
		return "", err
	}
	return data.Data.Id, nil

}

func GetSenderIdByAuth(userId string, auth string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/system/session/digital/%s/%s", baseUrl, userId, fixedLawyerId), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", auth)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Print the response body
	fmt.Println("Response body:", string(bodyBytes))

	// Create a new reader from the bytes for JSON decoding
	bodyReader := bytes.NewReader(bodyBytes)

	var data struct {
		Data struct {
			SenderId string `json:"senderId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bodyReader).Decode(&data); err != nil {
		return "", err
	}
	return data.Data.SenderId, nil
}

func ServeOnPort(port int) {
	http.HandleFunc("/system/message/list", proxyHandlerMessageList)
	http.HandleFunc("/logout", proxyHandlerLogout)
	log.Printf("Starting proxy server on port :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
