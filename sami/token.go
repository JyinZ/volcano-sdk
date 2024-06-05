package sami

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jyinz/volcano-sdk"
	"io"
	"net/http"
	"time"
)

// volcengine client wrapper
const (
	// sami open api info
	_Region  = "cn-north-1"
	_Version = "2021-07-27"
	_Service = "sami"
	_Path    = "/"

	_TokenVersion = "volc-auth-v1"
	_ResponseOK   = 20000000
)

type (
	GetTokenRequest struct {
		AppKey       string `json:"appkey"`
		TokenVersion string `json:"token_version"`
		Expiration   int64  `json:"expiration"`
	}

	GetTokenResponse struct {
		Code       int    `json:"code"`
		Msg        string `json:"msg"`
		StatusCode int32  `form:"status_code,required" json:"status_code,required" query:"status_code,required"`
		StatusText string `form:"status_text,required" json:"status_text,required" query:"status_text,required"`
		TaskId     string `form:"task_id,required" json:"task_id,required" query:"task_id,required"`
		Token      string `form:"token,required" json:"token,required" query:"token,required"`
		ExpiresAt  int64  `form:"expires_at,required" json:"expires_at,required" query:"expires_at,required"`
	}
)

type Token struct {
	volcano.OpenApi

	token     string
	expiresAt int64
}

// Expired returns true when token is expired.
func (tkn *Token) Expired() bool {
	// 提前1min判断过期，保证token使用过程时有效
	return tkn.expiresAt == 0 || time.Now().Unix() > tkn.expiresAt-60
}

// Refresh 刷新token
func (tkn *Token) Refresh(ctx context.Context, appKey string, expire int64) error {
	rsp, err := tkn.GetToken(ctx, GetTokenRequest{
		AppKey:     appKey,
		Expiration: expire,
	})
	if err != nil {
		return err
	}

	tkn.token = rsp.Token
	tkn.expiresAt = rsp.ExpiresAt

	return nil
}

// Token 获取当前token
func (tkn *Token) Token() string {
	return tkn.token
}

// Init 初始化一个token，一般用于测试
func (tkn *Token) Init(token string, expiredAt int64) {
	tkn.token = token
	tkn.expiresAt = expiredAt
}

func (tkn *Token) GetToken(ctx context.Context, gr GetTokenRequest) (*GetTokenResponse, error) {
	u := tkn.BuildUrl(_Path, _Version, "GetToken")

	gr.TokenVersion = _TokenVersion

	return tkn.post(ctx, u.String(), gr)
}

func (tkn *Token) post(ctx context.Context, url string, body any) (*GetTokenResponse, error) {
	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return do(tkn.Sign(req))
}

func do(req *http.Request) (*GetTokenResponse, error) {
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	defer rsp.Body.Close()

	rb, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var ret = new(GetTokenResponse)
	err = json.Unmarshal(rb, ret)
	if err != nil {
		return nil, fmt.Errorf("parse data failed: %w", err)
	}

	fmt.Println(string(rb))

	// 请求失败，解析错误原因
	if ret.StatusCode != _ResponseOK {
		return nil, fmt.Errorf("response error(%s): code = %d, desc = %s", rsp.Status, ret.Code, ret.Msg)
	}

	return ret, nil
}

func NewOpenApi(cfg volcano.Config) *Token {
	return &Token{
		OpenApi: volcano.OpenApi{
			Credentials: volcano.Credentials{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey,
				Service:         _Service,
				Region:          _Region,
			},
		},
	}
}
