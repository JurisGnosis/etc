package xunfei

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	STATUS_FIRST_FRAME    = 0
	STATUS_CONTINUE_FRAME = 1
	STATUS_LAST_FRAME     = 2
)

var wsParam *WsParam

func Init(appId string, apiSecret string, apiKey string) {
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

type WsParam struct {
	APPID     string
	APIKey    string
	APISecret string
	AudioFile string
	IatParams map[string]interface{}
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

func onMessage(ws *websocket.Conn) {
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		var response map[string]interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Println("json parse error:", err)
			continue
		}

		header := response["header"].(map[string]interface{})
		code := header["code"].(float64)
		status := header["status"].(float64)

		if code != 0 {
			log.Printf("请求错误：%d\n", int(code))
			ws.Close()
			break
		}

		if payload, ok := response["payload"].(map[string]interface{}); ok {
			resultText := payload["result"].(map[string]interface{})["text"].(string)
			decodedText, _ := base64.StdEncoding.DecodeString(resultText)
			fmt.Println("Result:", string(decodedText))
		}

		if status == 2 {
			ws.Close()
			break
		}
	}
}

func onOpen(ws *websocket.Conn, wsParam *WsParam) {
	file, err := os.Open(wsParam.AudioFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	const frameSize = 1280
	const interval = 40 * time.Millisecond
	status := STATUS_FIRST_FRAME

	for {
		audioBuffer := make([]byte, frameSize)
		n, err := file.Read(audioBuffer)
		if err != nil || n == 0 {
			if status == STATUS_CONTINUE_FRAME {
				status = STATUS_LAST_FRAME
			} else {
				break
			}
		}

		audio := base64.StdEncoding.EncodeToString(audioBuffer[:n])
		var message []byte

		switch status {
		case STATUS_FIRST_FRAME:
			status = STATUS_CONTINUE_FRAME
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
			ws.WriteMessage(websocket.TextMessage, message)
			break
		}

		ws.WriteMessage(websocket.TextMessage, message)
		time.Sleep(interval)
	}
	ws.Close()
}

func main() {

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+wsParam.APIKey)
	headers.Set("Content-Type", "application/json")

	ws, _, err := websocket.DefaultDialer.Dial(wsParam.createUrl(), headers)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer ws.Close()

	onOpen(ws, wsParam)
	onMessage(ws)
}
