package forward

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetUrl := r.URL.Query().Get("url")

	if strings.HasSuffix(targetUrl, "senderId=") {
		http.Error(w, "404", http.StatusBadRequest)
		return
	}

	// 创建新的请求
	targetUrl = strings.Replace(targetUrl, "https://lawyer.dlaws.cn:9900/", "http://47.107.101.100:9303/", 1)
	targetUrl = strings.Replace(targetUrl, "http://47.107.101.100:9304/", "http://47.107.101.100:9303/", 1)
	req, err := http.NewRequest(r.Method, targetUrl, r.Body)
	if err != nil {
		http.Error(w, "Error creating request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 复制来自客户端请求的头信息
	for key, value := range r.Header {
		req.Header[key] = value
	}

	// 发送请求给目标服务器
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置响应头及状态码
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// 写响应体到客户端
	_, err = w.Write(body)
	if err != nil {
		log.Println("Error writing response: " + err.Error())
	}
}

func ServeOnPort(port int) {
	http.HandleFunc("/", proxyHandler)
	log.Printf("Starting check on :%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
