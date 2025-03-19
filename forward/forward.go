package forward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"e.coding.net/Love54dj/weizhong/etc/ryconn"
)

// Constants
const (
	TestSessionID = "e5b21a57889541ffa01c6e387da971cd"
	ValidReferer  = "https://servicewechat.com/"
	FixedLawyerId = "132"
)

// Config holds the configuration for the proxy server
type Config struct {
	BaseURL      string
	LoginInfoURL string
	Routes       map[string]*RouteConfig
}

// RouteConfig holds the configuration for a specific route
type RouteConfig struct {
	TargetPath    string
	AuthValidator AuthValidator
	Middleware    []Middleware
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	baseURL := "http://47.107.101.100:9303"
	return &Config{
		BaseURL:      baseURL,
		LoginInfoURL: baseURL + "/system/loginInfo",
		Routes: map[string]*RouteConfig{
			"/system/message/list": {
				TargetPath:    "/system/message/list",
				AuthValidator: &MessageListAuthValidator{},
				Middleware: []Middleware{
					&MessageRequestHandler{},
				},
			},
			"/logout": {
				TargetPath:    "/logout",
				AuthValidator: &RefererAuthValidator{},
				Middleware:    []Middleware{},
			},
		},
	}
}

// AuthValidator defines the interface for authentication validation
type AuthValidator interface {
	Validate(r *http.Request) (string, error)
}

// Middleware defines the interface for request middleware
type Middleware interface {
	Process(w http.ResponseWriter, r *http.Request, auth string) error
}

// TokenAuthValidator validates authentication based on the Authorization header
type TokenAuthValidator struct{}

func (v *TokenAuthValidator) Validate(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("unauthorized: missing authorization header")
	}
	_, err := ryconn.AuthToMobile(auth)
	if err != nil {
		return "", err
	}
	return auth, nil
}

// RefererAuthValidator validates authentication based on the Referer header
type RefererAuthValidator struct{}

func (v *RefererAuthValidator) Validate(r *http.Request) (string, error) {
	referer := r.Header.Get("Referer")
	if !strings.HasPrefix(referer, ValidReferer) {
		return "", fmt.Errorf("unauthorized: invalid referer")
	}
	return "", nil // No auth token needed for this validator
}

// MessageListAuthValidator validates authentication for message list
type MessageListAuthValidator struct {
	TokenAuthValidator
}

func (v *MessageListAuthValidator) Validate(r *http.Request) (string, error) {
	// Special case for test session
	targetUrl := r.URL.String()
	if strings.Contains(targetUrl, TestSessionID) {
		return "", nil
	}

	// Otherwise use token validation
	return v.TokenAuthValidator.Validate(r)
}

// SenderIDValidator validates the sender ID in the request
type SenderIDValidator struct{}

func (m *SenderIDValidator) Process(w http.ResponseWriter, r *http.Request, auth string) error {
	// Check for invalid URL suffix
	if strings.HasSuffix(r.URL.String(), "senderId=") {
		return fmt.Errorf("invalid URL format")
	}

	// Get userId
	userId, err := GetIdByAuth(auth)
	if err != nil {
		return err
	}

	// Check senderId
	requestSenderId := r.URL.Query().Get("senderId")
	standardSenderId, err := GetSenderIdByAuth(userId, auth)
	if err != nil {
		return err
	}
	if requestSenderId != standardSenderId {
		return fmt.Errorf("invalid senderId")
	}

	return nil
}

// MessageRequestHandler handles both GET and POST requests for message list
type MessageRequestHandler struct{}

// Message represents the message request body structure
type Message struct {
	ID         int64  `json:"id"`
	MsgText    string `json:"msgText"`
	MsgType    int    `json:"msgType"`
	SenderID   string `json:"senderId"`
	SourceType int    `json:"sourceType"`
	UserID     int    `json:"userId"`
}

func (m *MessageRequestHandler) Process(w http.ResponseWriter, r *http.Request, auth string) error {
	// Handle GET requests using the existing SenderIDValidator logic
	if r.Method == http.MethodGet {
		validator := &SenderIDValidator{}
		return validator.Process(w, r, auth)
	}

	// Handle POST requests
	if r.Method == http.MethodPost {
		// Read and parse the request body
		var msg Message
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("error reading request body: %v", err)
		}

		// Create a new reader from the body for the forwarded request
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		// Unmarshal the JSON body
		if err := json.Unmarshal(body, &msg); err != nil {
			return fmt.Errorf("invalid JSON format: %v", err)
		}

		// Validate that senderId and userId
		if msg.SenderID == "" {
			return fmt.Errorf("missing senderId in request body")
		}

		if msg.UserID == 0 {
			return fmt.Errorf("missing or invalid userId in request body")
		}

		senderId, err := GetSenderIdByAuth(strconv.Itoa(msg.UserID), auth)
		if err != nil {
			return err
		}

		if senderId != msg.SenderID {
			return fmt.Errorf("invalid senderId")
		}

		return nil
	}

	return fmt.Errorf("unsupported HTTP method: %s", r.Method)
}

// ProxyServer represents the proxy server
type ProxyServer struct {
	Config *Config
	Client *http.Client
}

// NewProxyServer creates a new proxy server with the given configuration
func NewProxyServer(config *Config) *ProxyServer {
	if config == nil {
		config = DefaultConfig()
	}
	return &ProxyServer{
		Config: config,
		Client: &http.Client{},
	}
}

// ServeHTTP handles HTTP requests
func (s *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	routeConfig, exists := s.Config.Routes[path]
	if !exists {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Validate authentication
	auth, err := routeConfig.AuthValidator.Validate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Apply middleware
	for _, middleware := range routeConfig.Middleware {
		if err := middleware.Process(w, r, auth); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Forward the request
	s.forwardRequest(w, r, routeConfig.TargetPath)
}

// forwardRequest forwards the request to the target server
func (s *ProxyServer) forwardRequest(w http.ResponseWriter, r *http.Request, targetPath string) {
	// Build target URL
	targetURL := s.Config.BaseURL + targetPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create new request
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
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
	resp, err := s.Client.Do(req)
	if err != nil {
		log.Printf("Error making request to %s: %v", targetURL, err)
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

// GetIdByAuth retrieves the user ID using the authentication token
func GetIdByAuth(auth string) (string, error) {
	if auth == "" {
		return "", fmt.Errorf("empty authorization token")
	}

	config := DefaultConfig()
	req, err := http.NewRequest("GET", config.LoginInfoURL, nil)
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
			Id int `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bodyReader).Decode(&data); err != nil {
		return "", err
	}
	return strconv.Itoa(data.Data.Id), nil
}

// GetSenderIdByAuth retrieves the sender ID using the user ID and authentication token
func GetSenderIdByAuth(userId string, auth string) (string, error) {
	if userId == "" || auth == "" {
		return "", fmt.Errorf("empty userId or authorization token")
	}

	config := DefaultConfig()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/system/session/digital/%s/%s", config.BaseURL, userId, FixedLawyerId), nil)
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

// ServeOnPort starts the proxy server on the specified port
func ServeOnPort(port int) {
	proxyServer := NewProxyServer(nil) // Use default config

	// Register handlers for each route
	for route := range proxyServer.Config.Routes {
		http.Handle(route, proxyServer)
	}

	log.Printf("Starting proxy server on port :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// AddRoute adds a new route to the proxy server configuration
func AddRoute(config *Config, path string, targetPath string, authValidator AuthValidator, middleware ...Middleware) {
	if config.Routes == nil {
		config.Routes = make(map[string]*RouteConfig)
	}
	config.Routes[path] = &RouteConfig{
		TargetPath:    targetPath,
		AuthValidator: authValidator,
		Middleware:    middleware,
	}
}
