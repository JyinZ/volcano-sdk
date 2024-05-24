package vc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/jyinz/volcano-sdk"
	"github.com/jyinz/volcano-sdk/sami"
	"io"
	"net/http"
	"net/url"
)

const (
	_Host      = "sami.bytedance.com"
	_Namespace = "VoiceConversionStream"
)

type (
	AudioInfo struct {
		SampleRate int    `json:"sample_rate,omitempty"` // 音频采样率，大于等于8000, 小于等于48000
		Channel    int    `json:"channel,omitempty"`     // 音频通道数 1/2
		Format     string `json:"format,omitempty"`      // 音频编码格式，暂仅支持s16le
	}

	Extra struct {
		DownstreamAlign bool `json:"downstream_align"` // 是否要对齐每一帧长度（除了首包和尾包）
	}

	VoiceConversionRequest struct {
		Speaker     string    `json:"speaker"`         // 音色
		AudioInfo   AudioInfo `json:"audio_info"`      // 输入音频信息
		AudioConfig AudioInfo `json:"audio_config"`    // 输出音频信息
		Extra       *Extra    `json:"extra,omitempty"` // 补充信息
	}
)

func (vcr VoiceConversionRequest) wsMsg(ak, tkn string) []byte {
	pld, _ := json.Marshal(vcr)

	var wsMsg = sami.WebSocketRequest{
		Token:     tkn,
		Appkey:    ak,
		Namespace: _Namespace,
		Event:     sami.EventStartTask,
		Payload:   string(pld),
	}

	b, _ := json.Marshal(wsMsg)
	return b
}

type VoiceConversion struct {
	appKey string
	token  *sami.Token
}

func (c *VoiceConversion) Conversion(ctx context.Context, vcr VoiceConversionRequest, audio <-chan []byte, cb func([]byte)) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// 获取token
	if c.token.Expired() {
		err := c.token.Refresh(ctx, c.appKey)
		if err != nil {
			return fmt.Errorf("get token failed: %w", err)
		}
	}
	token := c.token.Token()

	u := url.URL{Scheme: "wss", Host: _Host, Path: "/api/v1/ws"}
	conn, rsp, err := websocket.DefaultDialer.DialContext(ctx, u.String(), http.Header{})
	if err != nil {
		if errors.Is(err, websocket.ErrBadHandshake) {
			defer rsp.Body.Close()
			b, _ := io.ReadAll(rsp.Body)
			return fmt.Errorf("%w, reason:%s", err, string(b))
		}
		return err
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, vcr.wsMsg(c.appKey, token))
	if err != nil {
		return fmt.Errorf("send config failed: %w", err)
	}

	// 等待第一个回包，校验服务端是否完成配置,TODO: 返回包错误处理
	//mt, msg, err := conn.ReadMessage()
	var wsRsp sami.WebSocketResponse
	err = conn.ReadJSON(&wsRsp)
	if err != nil {
		return fmt.Errorf("read first event failed: %w", err)
	}

	if !wsRsp.Started() {
		return fmt.Errorf("fisrt event mismatched")
	}

	go func() {
		defer func() {
			// 数据发送完毕时发送尾包
			fnsMsg, _ := json.Marshal(sami.WebSocketRequest{
				Token:     token,
				Appkey:    c.appKey,
				Namespace: _Namespace,
				Event:     sami.EventFinishTask,
			})
			_ = conn.WriteMessage(websocket.TextMessage, fnsMsg)
		}()

		step := 3200
		// 用于数据重组，平滑数据发送
		var buf bytes.Buffer
		for chunk := range audio {
			buf.Write(chunk)

			for buf.Len() > step {
				err := conn.WriteMessage(websocket.BinaryMessage, buf.Next(step))
				if err != nil {
					cancel(fmt.Errorf("send data failed :%w", err))
					return
				}
			}
		}

		err := conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
		if err != nil {
			cancel(fmt.Errorf("send data failed :%w", err))
			return
		}
	}()

	// 同步接收返回
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			cancel(fmt.Errorf("recv failed: %w", err))
			return context.Cause(ctx)
		}

		if mt == websocket.BinaryMessage {
			cb(msg)
			continue
		}

		err = json.Unmarshal(msg, &wsRsp)
		if err != nil {
			cancel(fmt.Errorf("parse data failed: %w", err))
			return context.Cause(ctx)
		}

		// TODO: 返回包错误处理
		_ = wsRsp.StatusCode

		if wsRsp.Data != nil {
			cb(wsRsp.Data)
		}

		if wsRsp.Finished() {
			break
		}
	}

	return nil
}

type Config struct {
	volcano.Config
	AppKey string `json:"app_key" yaml:"app_key"`
}

func New(cfg Config) *VoiceConversion {
	return &VoiceConversion{
		appKey: cfg.AppKey,
		token:  sami.NewOpenApi(cfg.Config),
	}
}
