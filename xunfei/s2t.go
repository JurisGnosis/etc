package xunfei

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	STATUS_FIRST_FRAME    = 0
	STATUS_CONTINUE_FRAME = 1
	STATUS_LAST_FRAME     = 2
	cache_dir             = "/tmp/"
)

// Global variables to hold the WebSocket parameters and a mutex for thread safety
var (
	wsParam                *WsParam
	mu                     sync.Mutex
	frameSize              int
	frameIntervalMillisecs int
)

type WsParam struct {
	APPID     string
	APIKey    string
	APISecret string
	AudioFile string
	IatParams map[string]interface{}
}

type WordInfo struct {
	Sc float64 `json:"sc"`
	W  string  `json:"w"`
}

type WordSegment struct {
	Bg int        `json:"bg"`
	Cw []WordInfo `json:"cw"`
}

type APIData struct {
	Sn int           `json:"sn"`
	Ls bool          `json:"ls"`
	Bg int           `json:"bg"`
	Ed int           `json:"ed"`
	Ws []WordSegment `json:"ws"`
}

func ExtractTextFromJSON(data string) (string, error) {
	var apiData APIData
	// 解析 JSON 数据
	err := json.Unmarshal([]byte(data), &apiData)
	if err != nil {
		return "", err
	}
	// 合并提取的文本内容
	var textContent string
	for _, segment := range apiData.Ws {
		for _, wordInfo := range segment.Cw {
			textContent += wordInfo.W
		}
	}
	return textContent, nil
}

func (wsParam *WsParam) createUrl() string {
	baseUrl := "wss://iat.cn-huabei-1.xf-yun.com/v1"
	host := "iat.cn-huabei-1.xf-yun.com"

	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\nGET /v1 HTTP/1.1", host, date)
	h := hmac.New(func() hash.Hash { return sha256.New() }, []byte(wsParam.APISecret))
	h.Write([]byte(signatureOrigin))
	signatureSha := base64.StdEncoding.EncodeToString(h.Sum(nil))

	authorizationOrigin := fmt.Sprintf(
		`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		wsParam.APIKey, signatureSha)

	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	v := url.Values{}
	v.Add("authorization", authorization)
	v.Add("date", date)
	v.Add("host", host)

	completeUrl := fmt.Sprintf("%s?%s", baseUrl, v.Encode())
	fmt.Println("websocket url :", completeUrl)
	return completeUrl
}

func Init(appId string, apiSecret string, apiKey string, frameSz int, frameIntervalMili int) {
	frameSize = frameSz
	frameIntervalMillisecs = frameIntervalMili
	wsParam = &WsParam{
		APPID:     appId,
		APISecret: apiSecret,
		APIKey:    apiKey,
		AudioFile: "path-to-your-audio-file",
		IatParams: map[string]interface{}{
			"domain": "slm", "language": "zh_cn", "accent": "mulacc", "result": map[string]string{"encoding": "utf8", "compress": "raw", "format": "json"},
		},
	}
}

func HandlerUpload(c *gin.Context) {
	// Check if the service is initialized
	if wsParam == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service not initialized"})
		return
	}

	file, err := c.FormFile("upload_wav")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error: " + err.Error()})
		return
	}

	var uploadedFile io.Reader

	if strings.HasSuffix(file.Filename, ".wav") {
		// generate random filename, and save file to `cache_path`
		cacheFile1 := path.Join(cache_dir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename))
		cacheFile2 := path.Join(cache_dir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename+".pcm"))
		err = c.SaveUploadedFile(file, cacheFile1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
			return
		}
		err = convertFileWavToPcm(cacheFile1, cacheFile2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert file: " + err.Error()})
			return
		}
		uploadedFile, err = os.Open(cacheFile2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error opening file: " + err.Error()})
			return
		}
	} else {
		uploadedFile00, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error opening file: " + err.Error()})
			return
		}
		defer uploadedFile00.Close()
		uploadedFile = uploadedFile00
	}

	// Read the uploaded file into a byte buffer
	audioBuffer := new(bytes.Buffer)
	if _, err := io.Copy(audioBuffer, uploadedFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading file: " + err.Error()})
		return
	}

	processAudio(audioBuffer.Bytes(), c) // Process the audio
}

func processAudio(audioData []byte, c *gin.Context) {
	mu.Lock()
	ws, _, err := websocket.DefaultDialer.Dial(wsParam.createUrl(), nil)
	mu.Unlock()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket connection error: " + err.Error()})
		return
	}
	defer ws.Close()

	onOpen(ws, audioData)
	retText := onMessage(ws)
	c.String(http.StatusOK, "{\"code\":200,\"msg\":\"Success\",\"data\":{\"ret\":1,\"data\":\"%s\"}}", retText)
}

func onOpen(ws *websocket.Conn, audioData []byte) {
	fmt.Println("socket open")
	fmt.Println(len(audioData))
	for offset := 0; offset < len(audioData); offset += frameSize {
		end := offset + frameSize
		if end > len(audioData) {
			end = len(audioData)
		}
		audioChunk := audioData[offset:end]
		audio := base64.StdEncoding.EncodeToString(audioChunk)
		var message []byte
		var status int
		// Determine the status
		switch {
		case offset == 0:
			status = STATUS_FIRST_FRAME
		case offset+frameSize >= len(audioData):
			status = STATUS_LAST_FRAME
		default:
			status = STATUS_CONTINUE_FRAME
		}
		switch status {
		case STATUS_FIRST_FRAME:
			message, _ = json.Marshal(map[string]interface{}{
				"header":    map[string]interface{}{"status": 0, "app_id": wsParam.APPID},
				"parameter": map[string]interface{}{"iat": wsParam.IatParams},
				"payload": map[string]interface{}{
					"audio": map[string]interface{}{
						"audio":       audio,
						"sample_rate": 16000,
						"encoding":    "raw",
					},
				},
			})
		case STATUS_CONTINUE_FRAME:
			message, _ = json.Marshal(map[string]interface{}{
				"header": map[string]interface{}{"status": 1, "app_id": wsParam.APPID},
				"payload": map[string]interface{}{
					"audio": map[string]interface{}{
						"audio":       audio,
						"sample_rate": 16000,
						"encoding":    "raw",
					},
				},
			})
		case STATUS_LAST_FRAME:
			message, _ = json.Marshal(map[string]interface{}{
				"header": map[string]interface{}{"status": 2, "app_id": wsParam.APPID},
				"payload": map[string]interface{}{
					"audio": map[string]interface{}{
						"audio":       audio,
						"sample_rate": 16000,
						"encoding":    "raw",
					},
				},
			})
		}
		ws.WriteMessage(websocket.TextMessage, message)
		time.Sleep(time.Duration(frameIntervalMillisecs) * time.Millisecond)
	}
}

type wsApiResponse struct {
	Header  wsApiHeader  `json:"header"`
	Payload wsApiPayload `json:"payload"`
}

type wsApiHeader struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	SID     string `json:"sid"`
	Status  int    `json:"status"`
}

type wsApiPayload struct {
	Result wsApiResult `json:"result"`
}

type wsApiResult struct {
	Compress string `json:"compress"`
	Encoding string `json:"encoding"`
	Format   string `json:"format"`
	Seq      int    `json:"seq"`
	Status   int    `json:"status"`
	Text     string `json:"text"`
}

func onMessage(ws *websocket.Conn) (retText string) {
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		response := wsApiResponse{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Println("json parse error:", err)
			continue
		}
		fmt.Println(string(message))

		header := response.Header
		code := header.Code
		// status := header.Status
		fmt.Println(header.Status)
		fmt.Println(response.Payload.Result.Status)

		if code != 0 {
			log.Printf("请求错误：%d\n", int(code))
			ws.Close()
			break
		}

		payload := response.Payload
		resultText := payload.Result.Text
		decodedText, _ := base64.StdEncoding.DecodeString(resultText)
		partialText, err := ExtractTextFromJSON(string(decodedText))
		if err != nil {
			log.Printf("解析失败: %s, %s", err.Error(), string(decodedText))
		}
		retText = retText + partialText
		// c.String(http.StatusOK, "Result: %s\n", string(decodedText))
		// if status == 2 {
		if response.Payload.Result.Status == 1 {
			ws.Close()
			break
		}
	}
	return
}
