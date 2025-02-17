package marker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

var baseUrl string
var authToken string

func Init(token string, listenAddr string, listenPort int) {
	baseUrl = fmt.Sprintf("http://%s:%d/convert", listenAddr, listenPort)
	authToken = token
}

func Pdf2Markdown(fileUrl string) (markdownText string, err error) {
	if authToken == "" || baseUrl == "" {
		slog.Error("marker-pdf needs initialization")
		return
	}
	tmpBody, _ := json.Marshal(map[string]string{"file_url": fileUrl})
	req, err := http.NewRequest(http.MethodPost, baseUrl, bytes.NewBuffer(tmpBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err == nil && req.Response.StatusCode != http.StatusOK {
		err = fmt.Errorf("API request failed: %s", body)
	}
	markdownText = string(body)
	return
}
