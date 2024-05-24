package mega

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jyinz/volcano-sdk"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	_Scheme = "https"
	_Host   = "openspeech.bytedance.com"
)

//go:generate stringer -type TrainingStatus
type TrainingStatus int

const (
	TrainingStatusNotFound TrainingStatus = iota
	TrainingStatusTraining
	TrainingStatusSuccess
	TrainingStatusFailed
	TrainingStatusActive
)

type (
	StatusRequest struct {
		AppID     string `json:"appid"`      // AppID of application
		SpeakerId string `json:"speaker_id"` // 唯一音色代号
	}

	Audio struct {
		AudioBytes  string `json:"audio_bytes"`  // 二进制音频字节，需对二进制音频进行base64编码
		AudioFormat string `json:"audio_format"` // 音频格式，pcm、m4a必传，其余可选
		Text        string `json:"text"`         // 可以让用户按照该文本念诵，服务会对比音频与该文本的差异。若差异过大会返回1109 WERError
	}

	UploadRequest struct {
		StatusRequest
		// Audios 音频格式支持：wav、mp3、ogg、m4a、aac、pcm，其中pcm仅支持24k 单通道
		//	目前限制单文件上传最大20MB, 每次最多上传1个音频文件
		Audios []Audio `json:"audios"` // 待训练的音频
		Source int     `json:"source"` // 固定值：2
	}
)

type (
	BaseResponse struct {
		StatusCode    int    `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	}

	UploadResponse struct {
		SpeakerId string `json:"speaker_id"`
	}

	StatusResponse struct {
		SpeakerId  string         `json:"speaker_id"`
		CreateTime int64          `json:"create_time"`
		Version    string         `json:"version"`
		DemoAudio  string         `json:"demo_audio"`
		Status     TrainingStatus `json:"status"`
	}
)

// TrainingCount 将返回中的训练版本转换为训练次数
func (r StatusResponse) TrainingCount() int {
	version := strings.TrimSuffix(r.Version, "V")
	vn, _ := strconv.Atoi(version)
	return vn
}

type OpenSpeech struct {
	AccessToken string
	AppID       string

	openapi *OpenApi
}

// Upload 上传音频素材进行训练
func (c *OpenSpeech) Upload(ctx context.Context, ur *UploadRequest) (string, error) {
	ur.AppID = c.AppID

	rb, err := c.do(ctx, "/api/v1/mega_tts/audio/upload", ur)
	if err != nil {
		return "", err
	}

	rsp := new(UploadResponse)

	err = json.Unmarshal(rb, rsp)
	if err != nil {
		return "", fmt.Errorf("parse data failed: %w", err)
	}

	return rsp.SpeakerId, nil
}

// Status 查询训练状态，仅在音色激活前可用，激活一段时间后使用将无法获取音色状态。
func (c *OpenSpeech) Status(ctx context.Context, mr *StatusRequest) (*StatusResponse, error) {
	mr.AppID = c.AppID

	rb, err := c.do(ctx, "/api/v1/mega_tts/status", mr)
	if err != nil {
		return nil, err
	}

	rsp := new(StatusResponse)
	err = json.Unmarshal(rb, rsp)
	if err != nil {
		return nil, fmt.Errorf("parse data failed: %w", err)
	}

	return rsp, nil
}

// ListSpeakers 获取音色列表，对open接口的简单封装调用
func (c *OpenSpeech) ListSpeakers(ctx context.Context, blr *BatchListMegaTTSTrainStatusRequest) (*SpeakerList, error) {
	blr.AppID = c.AppID

	return c.openapi.BatchListMegaTTSTrainStatus(ctx, blr)
}

// ActivateSpeakers 激活音色，即锁定使音色不再能够训练
func (c *OpenSpeech) ActivateSpeakers(ctx context.Context, ar *ActivateMegaTTSTrainStatusRequest) ([]SpeakerTrainStatus, error) {
	ar.AppID = c.AppID

	return c.openapi.ActivateMegaTTSTrainStatus(ctx, ar)
}

func (c *OpenSpeech) do(ctx context.Context, path string, body any) ([]byte, error) {
	u := url.URL{
		Scheme: _Scheme,
		Host:   _Host,
		Path:   path,
	}

	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer;"+c.AccessToken)
	req.Header.Set("Resource-Id", "volc.megatts.voiceclone")

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	defer rsp.Body.Close()

	rb, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 请求失败，解析错误原因
	if rsp.StatusCode != http.StatusOK {
		var ret BaseResponse
		err = json.Unmarshal(rb, &ret)
		if err != nil {
			return nil, fmt.Errorf("parse base response failed: %w", err)
		}

		return nil, fmt.Errorf("bad response: %w", NewError(ret.StatusCode))
	}

	return rb, nil
}

type (
	OpenAPIConfig struct {
		AccessKey string `json:"access_key" yaml:"access_key"`
		SecretKey string `json:"secret_key" yaml:"secret_key"`
	}

	Config struct {
		volcano.Config
		AccessToken string `json:"access_token" yaml:"access_token"`
		AppID       string `json:"app_id" yaml:"app_id"`
	}
)

func New(cfg Config) *OpenSpeech {
	return &OpenSpeech{
		AccessToken: cfg.AccessToken,
		AppID:       cfg.AppID,
		openapi:     NewOpenApi(cfg.Config),
	}
}
