package marker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

var baseUrl string
var authToken string
var singleThreadMutex sync.Mutex

func Init(host string, token string) {
	baseUrl = fmt.Sprintf("http://%s/convert", host)
	authToken = token
}

func Pdf2Markdown(fileUrl string) (markdownText string, err error) {
	if authToken == "" || baseUrl == "" {
		err = fmt.Errorf("marker-pdf needs initialization")
		slog.Error(err.Error())
		return
	}
	singleThreadMutex.Lock()
	defer singleThreadMutex.Unlock()
	tmpBody, _ := json.Marshal(map[string]string{"file_url": fileUrl})
	req, err := http.NewRequest(http.MethodPost, baseUrl, bytes.NewBuffer(tmpBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API request failed: %s", body)
	}
	markdownText = string(body)
	return
}
