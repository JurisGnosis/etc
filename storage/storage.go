package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var cosEndpoint string
var cosBaseUrl *cos.BaseURL
var cosClient *cos.Client
var timeoutSeconds = 100
var ctx = context.Background()

var filenameAccessCheck string = "timestampLastRegister"
var cosFrontendHost string
var cosFrontendScheme string

// 与 COS Bucket 绑定，建议不要修改
var usernameSalt string

func Init(SecretID string, SecretKey string, bucket string, region string, frontendHost string, frontendHttpsEnabled bool, sha1Key string) (err error) {
	cosEndpoint = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	cosFrontendHost = frontendHost
	if frontendHttpsEnabled {
		cosFrontendScheme = "https"
	} else {
		cosFrontendScheme = "http"
	}
	usernameSalt = sha1Key
	u, _ := url.Parse(cosEndpoint)
	cosBaseUrl = &cos.BaseURL{BucketURL: u}
	cosClient = cos.NewClient(cosBaseUrl, &http.Client{
		//设置超时时间
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &cos.AuthorizationTransport{
			//如实填写账号和密钥，也可以设置为环境变量
			SecretID:  SecretID,
			SecretKey: SecretKey,
		},
	})
	_, err = cosClient.Object.Put(ctx, filenameAccessCheck, io.NopCloser(strings.NewReader(time.Now().String())), nil)
	return
}

func CalcPath(userIdentifier string) string {
	hasher := sha1.New()
	hasher.Write([]byte(userIdentifier + usernameSalt))
	return hex.EncodeToString(hasher.Sum(nil))[:15]
}

func Upload(localFilePath string, targetFileName string, userIdentifier string) (downloadUrl string, err error) {
	userPath := CalcPath(userIdentifier)
	timePath := CalcPath(time.Now().String())[:4]
	// 上传文件
	_, err = cosClient.Object.PutFromFile(ctx, userPath+"/"+timePath+"/"+targetFileName, localFilePath, nil)
	if err == nil {
		downloadUrl = cosFrontendScheme + "://" + cosFrontendHost + "/" + path.Join(userPath, timePath, targetFileName)
	}
	return
}

func UploadRawContent(content string, targetFileName string, userIdentifier string) (downloadUrl string, err error) {
	userPath := CalcPath(userIdentifier)
	timePath := CalcPath(time.Now().String())[:4]
	// 上传文件
	_, err = cosClient.Object.Put(ctx, userPath+"/"+timePath+"/"+targetFileName, strings.NewReader(content), nil)
	if err == nil {
		downloadUrl = cosFrontendScheme + "://" + cosFrontendHost + "/" + path.Join(userPath, timePath, targetFileName)
	}
	return
}
