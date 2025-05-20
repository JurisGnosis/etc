package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

/*
 * Updated at 2025-05-20
 *
 * 模板类型消息 / 上传文件 / 语音消息 暂不支持，欢迎 PR
 *
 * 每个机器人发送的消息不能超过 20 条 / 分钟
 *
 */

const baseUrl = "https://qyapi.weixin.qq.com/cgi-bin/message/send"
const uploadUrl = "https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media"

type WechatArticle struct {
	Title       string `json:"title"`       // 标题，不超过 128 个字节，超过会自动截断
	Description string `json:"description"` // 描述，不超过 512 个字节，超过会自动截断
	Url         string `json:"url"`         // 点击后跳转的链接
	Picurl      string `json:"picurl"`      // 图片链接，支持JPG、PNG格式，较好的效果为大图 1068*455，小图 150*150
}

var apiKey string
var apiSendEndpoint string
var apiUploadEndpoint string
var cacheLog []string
var cacheLogMutex sync.Mutex
var ticker *time.Ticker
var fetchErrorFlag bool
var fetchErrorChan chan error

const flushInterval = 60 * time.Millisecond

func init() {
	ticker = time.NewTicker(flushInterval)
	go func() {
		for range ticker.C {
			cacheLogMutex.Lock()
			flushLog()
			cacheLogMutex.Unlock()
		}
	}()
}

func SetApiKey(key string) {
	apiKey = key
	apiSendEndpoint = fmt.Sprintf("%s?key=%s", baseUrl, key)
	apiUploadEndpoint = fmt.Sprintf("%s?key=%s", uploadUrl, key)
}

func SendLogSync(msg string) (err error) {
	cacheLogMutex.Lock()
	fetchErrorFlag = true
	cacheLog = append(cacheLog, msg)
	cacheLogMutex.Unlock()
	return <-fetchErrorChan
}

func SendLogAsync(msg string) {
	cacheLogMutex.Lock()
	cacheLog = append(cacheLog, msg)
	cacheLogMutex.Unlock()
}

func flushLog() {
	if len(cacheLog) == 0 {
		return
	}
	joinedLog := strings.Join(cacheLog, "\n\n")
	cacheLog = []string{}
	err := SendMarkdownMessage(joinedLog)
	if fetchErrorFlag {
		select {
		case fetchErrorChan <- err:
			fetchErrorFlag = false
		default:
		}
	}
}

func SendTextMessage(msg string, mentionedList []string, mentionedMobileList []string) (err error) {
	var req templateText
	req.MsgType = "text"
	req.Text.Content = msg
	req.Text.MentionedList = mentionedList
	req.Text.MentionedMobileList = mentionedMobileList
	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(apiSendEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return
}

func SendMarkdownMessage(msg string) (err error) {
	var req templateMarkdown
	req.MsgType = "markdown"
	req.Markdown.Content = msg
	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(apiSendEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return
}

func SendImageMessage(imageBin []byte) (err error) {
	var req templateImage
	req.MsgType = "image"
	req.Image.Base64 = base64.StdEncoding.EncodeToString(imageBin)
	var md5Hash = md5.Sum(imageBin)
	req.Image.Md5 = hex.EncodeToString(md5Hash[:])
	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(apiSendEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return
}

func SendNewsMessage(articles []WechatArticle) (err error) {
	var req templateNews
	req.MsgType = "news"
	req.News.Articles = articles
	jsonData, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(apiSendEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return
}

type templateText struct {
	MsgType string `json:"msgtype" binding:"required"` // text
	Text    struct {
		Content             string   `json:"content" binding:"required"`
		MentionedList       []string `json:"mentioned_list,omitempty"`        // ["zhangsan", "@all"]
		MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"` // ["13800138000", "@all"]
	} `json:"text"`
}

type templateMarkdown struct {
	MsgType  string `json:"msgtype" binding:"required"` // markdown
	Markdown struct {
		Content string `json:"content" binding:"required"`
	} `json:"markdown"`
}

type templateImage struct {
	MsgType string `json:"msgtype" binding:"required"` // image
	Image   struct {
		Base64 string `json:"base64" binding:"required"`
		Md5    string `json:"md5" binding:"required"`
	} `json:"image"`
}

type templateNews struct {
	MsgType string `json:"msgtype" binding:"required"` // news
	News    struct {
		Articles []WechatArticle `json:"articles"` // 图文消息，一个图文消息支持1到8条图文
	} `json:"news"`
}

type templateFile struct {
	MsgType string `json:"msgtype" binding:"required"` // file
	File    struct {
		MediaID string `json:"media_id" binding:"required"` // 文件id，通过下文的文件上传接口获取
	} `json:"file"`
}

type templateVoice struct {
	MsgType string `json:"msgtype" binding:"required"` // voice
	Voice   struct {
		MediaID string `json:"media_id" binding:"required"` // 语音id，通过下文的语音上传接口获取
	} `json:"voice"`
}

// TODO 文件上传接口
// 素材上传得到 media_id，该 media_id 仅三天内有效
// media_id 只能是对应上传文件的机器人可以使用
func UploadMedia() (mediaId string, err error) {
	// 使用 multipart/form-data POST 上传或语音，文件标识名为"media"
	// POST 的请求包中，form-data 中媒体文件标识，应包含有 filename、filelength、content-type 等信息
	/*
		请求示例
			POST https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=693a91f6-7xxx-4bc4-97a0-0ec2sifa5aaa&type=file HTTP/1.1
			Content-Type: multipart/form-data; boundary=-------------------------acebdf13572468
			Content-Length: 220

			---------------------------acebdf13572468
			Content-Disposition: form-data; name="media";filename="wework.txt"; filelength=6
			Content-Type: application/octet-stream

			mytext
			---------------------------acebdf13572468--
		返回数据
			{
				"errcode": 0,
				"errmsg": "ok",
				"type": "file",
				"media_id": "1234567890",
				"created_at": 1500000000
			}
		上传的文件限制
			普通文件（file）：20MB
			语音（voice）：2MB，播放长度不超过 60s，仅支持 AMR 格式
	*/
	return "not implemented", nil
}
