package pdf2doc

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const apiURL = "https://solution.wps.cn"
const contentType = "application/json"

var appID string
var appSecret string
var cachePath string
var notInitialized bool = true
var defaultError = errors.New("Please run pdf2doc.Init()")

func Init(WpsCachePath string, WpsAppid string, WpsAppsecret string) (err error) {
	appID = WpsAppid
	appSecret = WpsAppsecret
	cachePath = WpsCachePath
	// check if cachePath exists
	_, err = os.Stat(cachePath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(cachePath, os.FileMode(0766))
	}
	err = nil
	notInitialized = false
	return
}

// generateSignature 生成 API 请求的签名
func generateSignature(contentMd5Hex, date string) string {
	h := sha1.New()
	h.Write([]byte(appSecret + contentMd5Hex + contentType + date))
	signature := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("WPS-2:%s:%s", appID, signature)
}

func generateRequest(method string, requestBody any, url string) (req *http.Request, err error) {
	// JSON Marshal
	var body []byte
	if _, ok := requestBody.(string); !ok {
		body, err = json.Marshal(requestBody)
		if err != nil {
			err = fmt.Errorf("failed to marshal request body: %v", err)
			return
		}
	} else {
		body = []byte(requestBody.(string))
	}
	// 计算 Content-MD5
	contentMd5 := md5.Sum(body)
	contentMd5Hex := hex.EncodeToString(contentMd5[:])
	// 获取当前时间
	currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	// 生成签名
	signature := generateSignature(contentMd5Hex, currentTime)
	// 创建 HTTP 请求
	req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		err = fmt.Errorf("failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Md5", contentMd5Hex)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Date", currentTime)
	req.Header.Set("Authorization", signature)
	return
}

func IsExpired(err error) bool {
	return strings.Contains(err.Error(), "invalid docID")
}

func sendHttpRequest(req *http.Request) (responseBody []byte, err error) {
	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()
	// 读取响应
	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response: %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		if strings.Contains(string(responseBody), "invalid docID") {
			err = errors.New(string(responseBody))
		} else {
			err = fmt.Errorf("API request failed: %s", responseBody)
		}
		return
	}
	return
}

func Convert(pdfFilePath string) (TaskID string, err error) {
	if notInitialized {
		err = defaultError
		return
	}
	const selfPath string = "/api/developer/v1/office/pdf/convert/to/docx"
	// 构造请求体
	requestBody := map[string]string{
		"url": pdfFilePath,
		// "filename": "converted_document.docx", // 可以根据需要修改文件名
	}
	// 生成请求
	req, err := generateRequest(http.MethodPost, requestBody, apiURL+selfPath)
	if err != nil {
		return "", err
	}
	responseBody, err := sendHttpRequest(req)
	if err != nil {
		return "", err
	}
	// 解析响应（假设返回的是转换后文档的路径）
	var response convertResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return response.Data.TaskId, nil
}

func QueryResult(taskId string) (resp queryResponseData, err error) {
	if notInitialized {
		err = defaultError
		return
	}
	const selfPath string = "/api/developer/v1/tasks/convert/to/docx/"
	req, err := generateRequest(http.MethodGet, selfPath+taskId, apiURL+selfPath+taskId)
	if err != nil {
		err = fmt.Errorf("failed to generate request: %v", err)
		return
	}
	responseBody, err := sendHttpRequest(req)
	if err != nil {
		err = fmt.Errorf("failed to send request: %v", err)
		return
	}
	var response queryResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response: %v", err)
		return
	}
	resp = response.Data
	return
}

func DownloadResult(taskId string) (docFilePath string, err error) {
	if notInitialized {
		err = defaultError
		return
	}
	// check task status
	resp, err := QueryResult(taskId)
	if err != nil {
		return "", err
	}
	if resp.Status != 1 {
		return "", errors.New("任务未完成")
	}
	docFilePath = path.Join(cachePath, taskId+".docx")
	err = downloadFile(resp.DownloadURL, docFilePath)
	if err != nil {
		return "", err
	}
	return
}

func downloadFile(url string, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	// Get the response
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}
	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
