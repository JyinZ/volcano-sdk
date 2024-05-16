package sami

import (
	"encoding/json"
	"fmt"
	"github.com/volcengine/volc-sdk-golang/base"
	"net/http"
	"net/url"
	"time"
)

// volcengine client wrapper
const (
	// sami open api info
	DefaultRegion          = "cn-north-1"
	ServiceVersion20210727 = "2021-07-27"
	ServiceName            = "sami"
)

var (
	// DefaultInstance 默认的实例
	DefaultInstance = NewInstance()

	ServiceInfo = &base.ServiceInfo{
		Scheme:      "https",
		Timeout:     10 * time.Second,
		Host:        "open.volcengineapi.com",
		Header:      http.Header{"Accept": []string{"application/json"}},
		Credentials: base.Credentials{Region: "cn-north-1", Service: ServiceName},
	}

	ApiInfoList = map[string]*base.ApiInfo{
		"GetToken": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"GetToken"},
				"Version": []string{ServiceVersion20210727},
			},
		},
	}
)

var token string

func init() {
	// 定时清空 vc token
	go func() {
		ticker := time.NewTicker(59 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			token = ""
		}
	}()
}

func GetToken(accessKey, secretKey, appKey string) (string, error) {
	if token != "" {
		return token, nil
	}

	DefaultInstance.Client.ServiceInfo.Credentials = base.Credentials{Region: "cn-north-1", Service: "sami"}
	DefaultInstance.Client.SetAccessKey(accessKey)
	DefaultInstance.Client.SetSecretKey(secretKey)

	// Token parameters
	tokenVersion := "volc-auth-v1" // volc-auth-v1 volc-offline-auth-v1
	expiration := int64(3600)      // expiration in seconds
	resp := &GetTokenResponse{}

	//   2. Construct HTTP request with json body
	jsonBody := fmt.Sprintf(`{"appkey":"%v","token_version":"%v","expiration":%v}`, appKey, tokenVersion, expiration)
	status, err := DefaultInstance.commonHandlerJSON("GetToken", jsonBody, &resp)
	if err != nil {
		return "", err
	}

	if status < 200 || status >= 400 {
		return "", fmt.Errorf("http request fail. http code:%d", status)
	}

	token = resp.Token

	return token, nil
}

// Sami .
type Sami struct {
	Client *base.Client
}

// NewInstance new Sami client instance
func NewInstance() *Sami {
	instance := &Sami{}
	instance.Client = base.NewClient(ServiceInfo, ApiInfoList)
	instance.Client.ServiceInfo.Credentials.Service = ServiceName
	instance.Client.ServiceInfo.Credentials.Region = DefaultRegion
	return instance
}

func (p *Sami) commonHandler(api string, form url.Values, resp interface{}) (int, error) {
	respBody, statusCode, err := p.Client.Post(api, form, nil)
	fmt.Println(string(respBody))
	if err != nil {
		errMsg := err.Error()
		// business error will be shown in resp, request error should be nil here
		if errMsg[:3] != "api" {
			return statusCode, err
		}
	}

	if err := json.Unmarshal(respBody, resp); err != nil {
		return statusCode, err
	}
	return statusCode, nil
}

func (p *Sami) commonHandlerJSON(api string, jsonBody string, resp interface{}) (int, error) {
	respBody, statusCode, err := p.Client.Json(api, nil, jsonBody)
	fmt.Println(string(respBody))
	if err != nil {
		errMsg := err.Error()
		// business error will be shown in resp, request error should be nil here
		if errMsg[:3] != "api" {
			return statusCode, err
		}
	}

	if err := json.Unmarshal(respBody, resp); err != nil {
		return statusCode, err
	}
	return statusCode, nil
}
