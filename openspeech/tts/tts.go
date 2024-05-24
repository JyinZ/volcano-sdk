package tts

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
)

const (
	_Host = "openspeech.bytedance.com"
)

type (
	// APP 	应用相关配置
	APP struct {
		AppID   string `json:"appid"`   // 应用标识
		Token   string `json:"token"`   // 应用令牌，可传入任意非空值
		Cluster string `json:"cluster"` // 业务集群
	}

	// User 用户相关配置
	User struct {
		Uid string `json:"uid"` // 可传入任意非空值，传入值可以通过服务端日志追溯
	}

	// AudioConfig 音频相关配置
	AudioConfig struct {
		VoiceType       string  `json:"voice_type"`                 // 音色
		Encoding        string  `json:"encoding,omitempty"`         // 音频编码格式， wav(不支持流式) / pcm(默认) / ogg_opus / mp3
		CompressionRate int     `json:"compression_rate,omitempty"` // opus格式时编码压缩比， [1, 20]，默认为 1
		Rate            int     `json:"rate,omitempty"`             // 音频采样率，默认为 24000，可选8000，16000
		SpeedRatio      float64 `json:"speed_ratio,omitempty"`      // 语速，[0.2,3]，默认为1，通常保留一位小数即可
		VolumeRatio     float64 `json:"volume_ratio,omitempty"`     // 音量，[0.1, 3]，默认为1，通常保留一位小数即可
		PitchRatio      float64 `json:"pitch_ratio,omitempty"`      // 音高，[0.1, 3]，默认为1，通常保留一位小数即可
		Emotion         string  `json:"emotion,omitempty"`          // 情感风格
		Language        string  `json:"language,omitempty"`         // 语言类型
	}

	// Request 请求相关配置
	Request struct {
		Reqid           string `json:"reqid"`                      // 请求标识，需要保证每次调用传入值唯一，建议使用 UUID
		Text            string `json:"text"`                       // 文本，合成语音的文本，长度限制 1024 字节（UTF-8编码）。复刻音色没有此限制，但是HTTP接口有60s超时限制
		TextType        string `json:"text_type,omitempty"`        // 文本类型，plain / ssml, 默认为plain
		Operation       string `json:"operation"`                  // 操作，query（非流式，http只能query） / submit（流式）
		SilenceDuration int    `json:"silence_duration,omitempty"` // 句尾静音时长，单位为ms，默认为125
		WithFrontend    int    `json:"with_frontend,omitempty"`    // 时间戳相关，当with_frontend为1且frontend_type为unitTson的时候，返回音素级时间戳
		FrontendType    string `json:"frontend_type,omitempty"`    // unitTson
		WithTimestamp   int    `json:"with_timestamp,omitempty"`   // 新版时间戳参数，可用来替换with_frontend和frontend_type，可返回原文本的时间戳，而非TN后文本，即保留原文中的阿拉伯数字或者特殊符号等。注意：原文本中的多个标点连用或者空格依然会被处理，但不影响时间戳连贯性
		SplitSentence   int    `json:"split_sentence,omitempty"`   // 复刻音色语速优化,仅当使用复刻音色时设为1，可优化语速过快问题。有可能会导致时间戳多次返回。
		PureEnglishOpt  int    `json:"pure_english_opt,omitempty"` // 英文前端优化，当pure_english_opt为1的时候，中文音色读纯英文时可以正确处理文本中的阿拉伯数字
	}

	// SynRequest 合成请求体
	SynRequest struct {
		App     APP         `json:"app"`
		User    User        `json:"user"`
		Audio   AudioConfig `json:"audio"`
		Request Request     `json:"request"`
	}
)

func (sr *SynRequest) wsMsg(appID string) (msg []byte) {
	sr.App.AppID = appID
	sr.Request.Operation = "submit"
	input, _ := json.Marshal(sr)
	input = gzipCompress(input)

	// version: b0001 (4 bits)
	// header size: b0001 (4 bits)
	// message type: b0001 (Full client request) (4bits)
	// message type specific flags: b0000 (none) (4bits)
	// message serialization method: b0001 (JSON) (4 bits)
	// message compression: b0001 (gzip) (4bits)
	// reserved data: 0x00 (1 byte)
	var defaultHeader = []byte{0x11, 0x10, 0x11, 0x00}
	msg = make([]byte, 8+len(input))

	// 增加header
	copy(msg, defaultHeader)
	// body长度
	binary.BigEndian.PutUint32(msg[4:], uint32(len(input)))
	// 设置body
	copy(msg[8:], input)

	return msg
}

type (
	// Word 单字（符号）时间戳
	Word struct {
		Word      string  `json:"word"`
		StartTime float64 `json:"start_time"`
		EndTime   float64 `json:"end_time"`
		UnitType  string  `json:"unit_type"`
	}

	// Phoneme 因素时间戳
	Phoneme struct {
		Phone     string  `json:"phone"`
		StartTime float64 `json:"start_time"`
		EndTime   float64 `json:"end_time"`
	}

	// Frontend 文本时间戳
	Frontend struct {
		Words    []Word    `json:"words"`
		Phonemes []Phoneme `json:"phonemes"`
	}

	// Addition 额外数据
	Addition struct {
		Description string `json:"description"`
		Duration    string `json:"duration"`
		Frontend    string `json:"frontend"`
	}

	// SynResponse http一次合成返回结构体
	SynResponse struct {
		Reqid     string   `json:"reqid"`
		Code      int      `json:"code"`
		Operation string   `json:"operation"`
		Message   string   `json:"message"`
		Sequence  int      `json:"sequence"`
		Data      string   `json:"data"`
		Addition  Addition `json:"addition"`
	}
)

type (
	// FrontendMessage 流式请求时的结构体
	FrontendMessage struct {
		Duration string `json:"duration"`
		Frontend string `json:"frontend"`
	}

	// SynResult 流式合成结果
	SynResult struct {
		Chunk    []byte
		Frontend Frontend

		end bool
	}
)

func (ret *SynResult) parse(b []byte) (err error) {
	// 解析头部
	//protoVersion := b[0] >> 4
	headSize := b[0] & 0x0f
	messageType := b[1] >> 4
	messageTypeSpecificFlags := b[1] & 0x0f
	//serializationMethod := b[2] >> 4
	messageCompression := b[2] & 0x0f
	//reserve := b[3]
	//headerExtensions := b[4 : headSize*4]

	// payload
	payload := b[headSize*4:]
	switch messageType {
	case 0xb: // audio-only server response
		// no sequence number as ACK
		if messageTypeSpecificFlags == 0 {
		} else {
			sequenceNumber := int32(binary.BigEndian.Uint32(payload[0:4]))
			if sequenceNumber < 0 {
				ret.end = true
			}

			// 音频数据
			ret.Chunk = payload[8:]
		}
	case 0xc: // frontend server response
		payload = payload[4:]
		if messageCompression == 1 {
			payload = gzipDecompress(payload)
		}

		var fb FrontendMessage
		err = json.Unmarshal(payload, &fb)
		if err != nil {
			return
		}

		// 解析时间戳信息
		if fb.Frontend != "" {
			err = json.Unmarshal([]byte(fb.Frontend), &ret.Frontend)
			if err != nil {
				return
			}
		}

	case 0xf: // error message from server
		err = Error{
			Code: int32(binary.BigEndian.Uint32(payload[0:4])),
			Msg:  string(gzipDecompress(payload[8:])),
		}
	}

	return
}

type TTS struct {
	AccessToken string
	AppID       string
}

// Synthesize 在线流式合成
func (c *TTS) Synthesize(ctx context.Context, sr SynRequest, cb func(SynResult)) error {
	u := url.URL{Scheme: "wss", Host: _Host, Path: "/api/v1/tts/ws_binary"}
	header := http.Header{"Authorization": []string{fmt.Sprintf("Bearer;%s", c.AccessToken)}}

	conn, rsp, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		if errors.Is(err, websocket.ErrBadHandshake) {
			defer rsp.Body.Close()
			b, _ := io.ReadAll(rsp.Body)
			return fmt.Errorf("%w, reason:%s", err, string(b))
		}
		return err
	}
	defer conn.Close()

	// 发送请求
	err = conn.WriteMessage(websocket.BinaryMessage, sr.wsMsg(c.AppID))
	if err != nil {
		return fmt.Errorf("write message fail, err: %s", err.Error())
	}

	// 接收返回
	for {
		var message []byte
		_, message, err = conn.ReadMessage()
		if err != nil {
			// log error
			return err
		}

		var ret SynResult
		err = ret.parse(message)
		if err != nil {
			return fmt.Errorf("parse message failed: %w", err)
		}

		cb(ret)
		if ret.end {
			break
		}
	}

	return nil
}

type Config struct {
	AccessToken string `json:"access_token" yaml:"access_token"`
	AppID       string `json:"app_id" yaml:"app_id"`
}

func New(cfg Config) *TTS {
	return &TTS{
		AccessToken: cfg.AccessToken,
		AppID:       cfg.AppID,
	}
}

func gzipCompress(input []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(input)
	_ = w.Close()
	return b.Bytes()
}

func gzipDecompress(input []byte) []byte {
	r, _ := gzip.NewReader(bytes.NewBuffer(input))
	out, _ := io.ReadAll(r)
	_ = r.Close()
	return out
}
