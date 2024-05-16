package volcano

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// open api info
const (
	// 请求接口信息
	_Service = "speech_saas_prod"
	_Region  = "cn-north-1"
	_Version = "2023-11-07"
	_Path    = "/"
)

type SpeakerState = string

const (
	SpeakerStateUnknown   SpeakerState = "Unknown"   // 未找到对应SpeakerID的记录、ps: 未使用过也为该状态
	SpeakerStateTraining               = "Training"  // 声音复刻中（长时间处于复刻中状态请联系TODO）
	SpeakerStateSuccess                = "Success"   // 声音复刻成功，可以进行启动(update)操作
	SpeakerStateActive                 = "Active"    // 使用中
	SpeakerStateExpired                = "Expired"   // 火山控制台实例已过期或账号欠费
	SpeakerStateReclaimed              = "Reclaimed" // 火山控制台实例已回收
)

type (
	LiseMegaTTSTrainStatusRequest struct {
		AppID      string   `json:"AppID"`                // AppID of application
		SpeakerIDs []string `json:"SpeakerIDs,omitempty"` // List of speaker IDs; if empty, resp will return all speakerIDs from given appID
	}

	BatchListMegaTTSTrainStatusRequest struct {
		LiseMegaTTSTrainStatusRequest
		PageNumber int    `json:"PageNumber,omitempty"` // Page number requested, larger than 0, default to 1
		PageSize   int    `json:"PageSize,omitempty"`   // Page size requested, must be in range [1, 100], default to 10
		NextToken  string `json:"NextToken,omitempty"`  // String acquired from last response; if not nil, overwrite page number and size settings
		MaxResults int    `json:"MaxResults,omitempty"` // MaxResults of results returned; must be used together with NextToken; if not nil, must be in range [1, 100], default to 10
	}

	ActivateMegaTTSTrainStatusRequest struct {
		AppID      string   `json:"AppID"`      // AppID of application
		SpeakerIDs []string `json:"SpeakerIDs"` // List of speaker IDs
	}
)

type (
	Metadata struct {
		RequestId string `json:"RequestId"`
		Action    string `json:"Action"`
		Version   string `json:"Version"`
		Service   string `json:"Service"`
		Region    string `json:"Region"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	}

	SpeakerTrainStatus struct {
		CreateTime  int64        `json:"CreateTime,omitempty"`
		DemoAudio   string       `json:"DemoAudio,omitempty"`
		InstanceNO  string       `json:"InstanceNO,omitempty"`
		IsActivable bool         `json:"IsActivable,omitempty"`
		SpeakerID   string       `json:"SpeakerID"`
		State       SpeakerState `json:"State"`
		Version     string       `json:"Version,omitempty"`
	}

	SpeakerList struct {
		AppID      string               `json:"AppID"`
		TotalCount int                  `json:"TotalCount"`
		NextToken  string               `json:"NextToken"`
		PageNumber int                  `json:"PageNumber"`
		PageSize   int                  `json:"PageSize"`
		Statuses   []SpeakerTrainStatus `json:"Statuses"`
	}

	ListResponse struct {
		ResponseMetadata Metadata    `json:"ResponseMetadata"`
		Result           SpeakerList `json:"Result,omitempty"`
	}

	ActiveResponse = ListResponse
)

// LiseMegaTTSTrainStatus 查询已购买的音色状态
// 如果SpeakerIDs为空则返回账号的AppID下所有的列表（有最大值限制1000）
// 如果SpeakerIDs不为空则返回对应的结果，且结果总是包含输入的SpeakerID（即使查询不到它） (ps：经测试，若包含错误的SpeakerID会返回错误)
func (c *OpenApi) LiseMegaTTSTrainStatus(ctx context.Context, lr *LiseMegaTTSTrainStatusRequest) (*ListResponse, error) {
	u := c.buildUrl("ListMegaTTSTrainStatus")

	return c.post(ctx, u, lr)
}

// BatchListMegaTTSTrainStatus 查询已购买的音色状态；相比ListMegaTTSTrainStatus 增加了分页相关参数和返回；支持使用token和声明页数两种分页方式
// 分页token在最后一页为空；分页token采用私有密钥进行加密
func (c *OpenApi) BatchListMegaTTSTrainStatus(ctx context.Context, blr *BatchListMegaTTSTrainStatusRequest) (*ListResponse, error) {
	u := c.buildUrl("BatchListMegaTTSTrainStatus")

	return c.post(ctx, u, blr)
}

// ActivateMegaTTSTrainStatus 激活 (activate) SpeakerID。目前已无需调用该接口即可进行tts合成。调用该接口后将无法继续训练，无论是否还有剩余的训练次数。
// 如果输入的音色列表中有一个或多个音色不能被激活，或找不到它的记录，那么所有的音色都不会被激活，并且会返回错误 OperationDenied.InvalidSpeakerID
// 如果输入的音色列表为空，那么返回字段不合法的错误OperationDenied.InvalidParameter
// 距离音色可被访问可能会有分钟级别延迟
func (c *OpenApi) ActivateMegaTTSTrainStatus(ctx context.Context, ar *ActivateMegaTTSTrainStatusRequest) (*ActiveResponse, error) {
	u := c.buildUrl("ActivateMegaTTSTrainStatus")

	return c.post(ctx, u, ar)
}

func (c *OpenApi) buildUrl(action string) string {
	u := url.URL{
		Scheme: _Scheme,
		Host:   _Host,
		Path:   _Path,
		RawQuery: url.Values{
			"Version": []string{_Version},
			"Action":  []string{action},
		}.Encode(),
	}

	return u.String()
}

func (c *OpenApi) post(ctx context.Context, url string, body any) (*ListResponse, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return do(c.Sign(req))
}

func do[T ListResponse](req *http.Request) (*T, error) {
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response code: %s", rsp.Status)
	}

	rb, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var ret = new(T)
	err = json.Unmarshal(rb, ret)
	if err != nil {
		return nil, fmt.Errorf("parse data failed: %w", err)
	}

	return ret, nil
}

func NewOpenApi(cfg Config) OpenApi {
	return OpenApi{
		Credentials: Credentials{
			AccessKeyID:     cfg.AccessKey,
			SecretAccessKey: cfg.SecretKey,
			Service:         _Service,
			Region:          _Region,
		},
	}
}
