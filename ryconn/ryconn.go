package ryconn

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var loginUrl string = "https://lawyer.dlaws.cn:9900/api/lawyer/master/system/loginInfo"

func Init(loginInfoUrl string) {
	loginUrl = loginInfoUrl
}

func AuthToMobile(autoToken string) (mobile string, err error) {
	// access loginUrl with autoToken as header: Authorization
	client := &http.Client{}
	req, err := http.NewRequest("GET", loginUrl, nil)
	if err != nil {
		return
	}
	if strings.Contains(autoToken, "Bearer ") {
		req.Header.Add("Authorization", autoToken)
	} else {
		req.Header.Add("Authorization", "Bearer "+autoToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	// parse response body
	respStruct := RuoyiResponse{}
	err = json.NewDecoder(resp.Body).Decode(&respStruct)
	if err != nil {
		return
	}
	if respStruct.Code != 200 {
		err = errors.New(respStruct.Msg)
		return
	}
	mobile = respStruct.Data.Mobile
	if mobile == "" {
		err = errors.New("Not logged in.")
		return
	}
	return
}
