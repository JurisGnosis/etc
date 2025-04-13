package difystorage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDifyStorage tests the Dify storage functionality
// This test will be skipped if the required environment variables are not set
func TestDifyStorage(t *testing.T) {
	// Get API URL and key from environment variables
	apiURL := os.Getenv("DIFY_API_URL")
	apiKey := os.Getenv("DIFY_API_KEY")
	sampleFilePath := os.Getenv("DIFY_SAMPLE_FILE")

	// Skip test if any of the required environment variables are not set
	if apiURL == "" || apiKey == "" || sampleFilePath == "" {
		t.Skip("Skipping test: DIFY_API_URL, DIFY_API_KEY, and DIFY_SAMPLE_FILE environment variables must be set")
	}

	// Test Init
	t.Run("Init", func(t *testing.T) {
		err := Init(apiURL, apiKey)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
	})

	// Test Upload
	t.Run("Upload", func(t *testing.T) {
		// Check if the sample file exists
		if _, err := os.Stat(sampleFilePath); os.IsNotExist(err) {
			t.Fatalf("Sample file does not exist: %s", sampleFilePath)
		}

		// Upload the sample file
		fileName := filepath.Base(sampleFilePath)
		fileId, err := Upload(sampleFilePath, fileName, "test-user")
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		// Verify fileId is not empty
		if fileId == "" {
			t.Fatal("Upload returned empty fileId")
		}

		t.Logf("File uploaded successfully with ID: %s", fileId)
	})

	// Test UploadRawContent
	t.Run("UploadRawContent", func(t *testing.T) {
		// Create test content
		testContent := "This is a test content for Dify storage"
		testFileName := "test-content.txt"

		// Upload the content
		fileId, err := UploadRawContent(testContent, testFileName, "test-user")
		if err != nil {
			t.Fatalf("UploadRawContent failed: %v", err)
		}

		// Verify fileId is not empty
		if fileId == "" {
			t.Fatal("UploadRawContent returned empty fileId")
		}

		t.Logf("Content uploaded successfully with ID: %s", fileId)
	})
}

// TestDifyStorageWithFlags tests the Dify storage functionality with command-line flags
// This allows for more flexible testing with custom parameters
func TestDifyStorageWithFlags(t *testing.T) {
	// Get API URL and key from command-line flags or environment variables
	apiURL := getTestParam("DIFY_API_URL", "")
	apiKey := getTestParam("DIFY_API_KEY", "")
	sampleFilePath := getTestParam("DIFY_SAMPLE_FILE", "")

	// Skip test if any of the required parameters are not set
	if apiURL == "" || apiKey == "" || sampleFilePath == "" {
		t.Skip("Skipping test: API URL, API key, and sample file path must be provided")
	}

	// Test Init
	t.Run("Init", func(t *testing.T) {
		err := Init(apiURL, apiKey)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
	})

	// Test Upload
	t.Run("Upload", func(t *testing.T) {
		// Check if the sample file exists
		if _, err := os.Stat(sampleFilePath); os.IsNotExist(err) {
			t.Fatalf("Sample file does not exist: %s", sampleFilePath)
		}

		// Upload the sample file
		fileName := filepath.Base(sampleFilePath)
		fileId, err := Upload(sampleFilePath, fileName, "test-user")
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		// Verify fileId is not empty
		if fileId == "" {
			t.Fatal("Upload returned empty fileId")
		}

		t.Logf("File uploaded successfully with ID: %s", fileId)
	})

	// Test UploadRawContent
	t.Run("UploadRawContent", func(t *testing.T) {
		// Create test content
		testContent := "This is a test content for Dify storage"
		testFileName := "test-content.txt"

		// Upload the content
		fileId, err := UploadRawContent(testContent, testFileName, "test-user")
		if err != nil {
			t.Fatalf("UploadRawContent failed: %v", err)
		}

		// Verify fileId is not empty
		if fileId == "" {
			t.Fatal("UploadRawContent returned empty fileId")
		}

		t.Logf("Content uploaded successfully with ID: %s", fileId)
	})
}

// Helper function to get test parameters from environment variables or flags
func getTestParam(envName string, defaultValue string) string {
	// First check environment variable
	value := os.Getenv(envName)
	if value != "" {
		return value
	}

	// If not found in environment, return default value
	return defaultValue
}

// Example of how to run the tests:
// 
// 1. Using environment variables:
//    DIFY_API_URL="https://dify.longhua.dlaws.cn" DIFY_API_KEY="your-api-key" DIFY_SAMPLE_FILE="/path/to/sample.jpg" go test -v
//
// 2. Using go test with -run flag to run specific tests:
//    DIFY_API_URL="https://dify.longhua.dlaws.cn" DIFY_API_KEY="your-api-key" DIFY_SAMPLE_FILE="/path/to/sample.jpg" go test -v -run TestDifyStorage
