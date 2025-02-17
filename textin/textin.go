package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/go-querystring/query"
)

var textin = &TextinOcr{
	AppID:     "",
	AppSecret: "",
	Host:      "https://api.textin.com",
}

var options = Options{
	PageStart:   0,
	PageCount:   1000, // 解析1000页
	TableFlavor: "md",
	ParseMode:   "scan", // 设置为scan模式
	Dpi:         144,    // 分辨率为144 dpi
	PageDetails: 0,      // 不包含页面细节信息
}

type TextinOcr struct {
	AppID     string
	AppSecret string
	Host      string
}

type Options struct {
	PdfPwd            string `url:"pdf_pwd,omitempty"`
	Dpi               int    `url:"dpi,omitempty"`
	PageStart         int    `url:"page_start"`
	PageCount         int    `url:"page_count"`
	ApplyDocumentTree int    `url:"apply_document_tree,omitempty"`
	MarkdownDetails   int    `url:"markdown_details,omitempty"`
	TableFlavor       string `url:"table_flavor,omitempty"`
	GetImage          string `url:"get_image,omitempty"`
	ParseMode         string `url:"parse_mode,omitempty"`
	PageDetails       int    `url:"page_details,omitempty"`
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  struct {
		Markdown string `json:"markdown"`
	} `json:"result"`
}

func getFileContent(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (ocr *TextinOcr) recognizePDF2MD(image []byte, options Options, isUrl bool) (*http.Response, error) {
	url := ocr.Host + "/ai/service/v1/pdf_to_markdown"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(image))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-ti-app-id", ocr.AppID)
	req.Header.Set("x-ti-secret-code", ocr.AppSecret)
	if isUrl {
		req.Header.Set("Content-Type", "text/plain")
	} else {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	q, _ := query.Values(options)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	return client.Do(req)
}

func writeFile(content, filePath string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

func Init(appId string, appSecret string) {
	textin.AppID = appId
	textin.AppSecret = appSecret
}

func Pdf2MarkdownFromLocal(filePath string) (markdownText string, err error) {
	if textin.AppID == "" || textin.AppSecret == "" {
		panic("AppID or AppSecret is empty")
	}
	// 示例 1：传输文件
	image, err := getFileContent(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	start := time.Now()
	resp, err := textin.recognizePDF2MD(image, options, false)
	if err != nil {
		fmt.Println("Error recognizing PDF:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request time: ", time.Duration(time.Now().Sub(start)))

	var jsonData Response
	if err = json.NewDecoder(resp.Body).Decode(&jsonData); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}
	markdownText = jsonData.Result.Markdown
	return
}

func Pdf2MarkdownFromUrl(url string) (markdownText string, err error) {
	if textin.AppID == "" || textin.AppSecret == "" {
		panic("AppID or AppSecret is empty")
	}
	// 示例 2：传输 URL
	start := time.Now()
	resp, err := textin.recognizePDF2MD([]byte(url), options, true)
	if err != nil {
		fmt.Println("Error recognizing PDF:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Request time: ", time.Duration(time.Now().Sub(start)))

	var jsonData Response
	if err = json.NewDecoder(resp.Body).Decode(&jsonData); err != nil {
		fmt.Println("Error decoding response:", err)
		return
	}
	markdownText = jsonData.Result.Markdown
	return
}
