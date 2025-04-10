package difystorage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Dify API configuration
var difyAPIURL string
var difyAPIKey string

// DifyFileResponse represents the response from Dify file upload API
type DifyFileResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	CreatedBy string `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
}

// Init initializes the Dify storage with the provided API URL and API key
func Init(apiURL string, apiKey string) (err error) {
	slog.Info("initializing Dify storage", "apiURL", apiURL)

	// Validate API URL format
	if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
		return errors.New("API URL must start with http:// or https://")
	}

	// Ensure URL doesn't end with a slash
	if strings.HasSuffix(apiURL, "/") {
		apiURL = strings.TrimSuffix(apiURL, "/")
	}

	// Validate API key is not empty
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	// Test connection to Dify API
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL+"/v1", nil)
	if err != nil {
		slog.Error("failed to create test request", "error", err)
		return fmt.Errorf("failed to create test request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to connect to Dify API", "error", err)
		return fmt.Errorf("failed to connect to Dify API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Dify API connection test failed", "statusCode", resp.StatusCode)
		return fmt.Errorf("Dify API connection test failed with status code: %d", resp.StatusCode)
	}

	// Set global variables
	difyAPIURL = apiURL
	difyAPIKey = apiKey

	slog.Info("Dify storage initialized successfully")
	return nil
}

// CalcPath is kept for compatibility but not used in Dify storage
// It returns an empty string as it's not needed
func CalcPath(userIdentifier string) string {
	// Not used in Dify storage, but kept for compatibility
	_ = userIdentifier // Send to /dev/null to avoid lint errors
	return ""
}

// GetStoragePath is kept for compatibility but not used in Dify storage
// It returns an empty string as it's not needed
func GetStoragePath(targetFileName string, userIdentifier string) string {
	// Not used in Dify storage, but kept for compatibility
	_ = targetFileName // Send to /dev/null to avoid lint errors
	_ = userIdentifier // Send to /dev/null to avoid lint errors
	return ""
}

// Upload uploads a file from a local path to Dify storage
func Upload(localFilePath string, targetFileName string, userIdentifier string) (fileId string, err error) {
	slog.Info("uploading file to Dify", "localFilePath", localFilePath, "targetFileName", targetFileName, "userIdentifier", userIdentifier)

	// Check if source file exists
	if _, err = os.Stat(localFilePath); os.IsNotExist(err) {
		slog.Error("source file does not exist", "path", localFilePath)
		return "", errors.New("source file does not exist")
	}

	// Open the file
	file, err := os.Open(localFilePath)
	if err != nil {
		slog.Error("failed to open source file", "path", localFilePath, "error", err)
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	// Create a new multipart buffer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file field
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(targetFileName))
	if err != nil {
		slog.Error("failed to create form file", "error", err)
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to form field
	if _, err = io.Copy(fileWriter, file); err != nil {
		slog.Error("failed to copy file content", "error", err)
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add user field if provided
	if userIdentifier != "" {
		if err = writer.WriteField("user", userIdentifier); err != nil {
			slog.Error("failed to add user field", "error", err)
			return "", fmt.Errorf("failed to add user field: %w", err)
		}
	}

	// Close the multipart writer
	if err = writer.Close(); err != nil {
		slog.Error("failed to close multipart writer", "error", err)
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", difyAPIURL+"/v1/files/upload", body)
	if err != nil {
		slog.Error("failed to create HTTP request", "error", err)
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+difyAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to send HTTP request", "error", err)
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("Dify API returned error", "statusCode", resp.StatusCode, "response", string(bodyBytes))
		return "", fmt.Errorf("Dify API returned error with status code: %d", resp.StatusCode)
	}

	// Parse response
	var fileResponse DifyFileResponse
	if err = json.NewDecoder(resp.Body).Decode(&fileResponse); err != nil {
		slog.Error("failed to parse API response", "error", err)
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	slog.Info("file uploaded successfully to Dify", "fileId", fileResponse.ID, "fileName", fileResponse.Name)
	return fileResponse.ID, nil
}

// UploadRawContent uploads raw text content to Dify storage
func UploadRawContent(plainText string, targetFileName string, userIdentifier string) (fileId string, err error) {
	slog.Info("uploading raw content to Dify", "targetFileName", targetFileName, "userIdentifier", userIdentifier)

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "dify-upload-*"+filepath.Ext(targetFileName))
	if err != nil {
		slog.Error("failed to create temporary file", "error", err)
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath) // Clean up temp file when done

	// Write content to the temporary file
	if _, err = tempFile.WriteString(plainText); err != nil {
		tempFile.Close()
		slog.Error("failed to write content to temporary file", "error", err)
		return "", fmt.Errorf("failed to write content to temporary file: %w", err)
	}
	tempFile.Close()

	// Use the Upload function to upload the temporary file
	fileId, err = Upload(tempFilePath, targetFileName, userIdentifier)
	if err != nil {
		slog.Error("failed to upload content", "error", err)
		return "", err
	}

	slog.Info("content uploaded successfully to Dify", "fileId", fileId)
	return fileId, nil
}
