package openspeech

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"

	log "digital-robot3d/pkg/log"
)

const (
	path             = "/api/v1/tts/ws_binary"
	optSubmit string = "submit"
)

// version: b0001 (4 bits)
// header size: b0001 (4 bits)
// message type: b0001 (Full client request) (4bits)
// message type specific flags: b0000 (none) (4bits)
// message serialization method: b0001 (JSON) (4 bits)
// message compression: b0001 (gzip) (4bits)
// reserved data: 0x00 (1 byte)
var defaultHeader = []byte{0x11, 0x10, 0x11, 0x00}

type Config struct {
	Addr  string
	AppID string
	Token string
}

type TTS struct {
	config Config
}

func NewTTS(addr, appID, token string) *TTS {
	return &TTS{
		config: Config{
			Addr:  addr,
			AppID: appID,
			Token: token,
		},
	}
}

type Params struct {
	Pitch      float32
	Speed      float32
	VoiceType  string
	SampleRate uint32
	Volume     float32
	Cluster    string
}

func (tts *TTS) Synthesize(requestID, text string, params Params, callback func(SynResp)) error {
	input := tts.setupInput(requestID, text, optSubmit, params)
	log.Info("[bytedance_tts]", "input params", string(input))

	var clientRequest = make([]byte, len(defaultHeader))
	{
		input = gzipCompress(input) // 压缩
		payloadSize := len(input)
		payloadArr := make([]byte, 4)
		binary.BigEndian.PutUint32(payloadArr, uint32(payloadSize))

		copy(clientRequest, defaultHeader)
		clientRequest = append(clientRequest, payloadArr...)
		clientRequest = append(clientRequest, input...)
	}

	u := tts.getWSSUrl()
	header := tts.getHeader()
	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("dial err: %s", err.Error())
	}
	defer c.Close()

	err = c.WriteMessage(websocket.BinaryMessage, clientRequest)
	if err != nil {
		return fmt.Errorf("write message fail, err: %s", err.Error())
	}

	for {
		var message []byte
		_, message, err = c.ReadMessage()
		if err != nil {
			// log error
			log.Error("[bytedance_tts]read message fail", "err:", err.Error())
			return err
		}

		var (
			resp SynResp
		)
		resp, err = tts.parseResponse(message)
		if err != nil {
			// log error
			log.Error(fmt.Sprintf("[bytedance_tts]parse response fail, err:%v", err))
			return err
		}

		callback(resp)
		if resp.IsLast {
			break
		}
	}

	return nil
}

func (tts *TTS) setupInput(requestID, text, opt string, p Params) []byte {
	b, _ := json.Marshal(p)

	log.Info("[bytedance_tts]", "setupInput p", string(b))

	//reqID := uuid.Must(uuid.NewV4(), nil).String()
	params := make(map[string]map[string]interface{})
	params["app"] = make(map[string]interface{})
	//平台上查看具体appid
	params["app"]["appid"] = tts.config.AppID
	params["app"]["token"] = "access_token"
	//平台上查看具体集群名称
	params["app"]["cluster"] = p.Cluster // 用 语音合成里面提供的音色 cluster=volcano_tts; 用声音复刻的音色cluster=volcano_mega
	params["user"] = make(map[string]interface{})
	params["user"]["uid"] = tts.config.AppID
	params["audio"] = make(map[string]interface{})
	params["audio"]["voice_type"] = p.VoiceType
	params["audio"]["rate"] = p.SampleRate
	params["audio"]["bit"] = 16
	params["audio"]["encoding"] = "pcm"
	params["audio"]["speed_ratio"] = p.Speed
	params["audio"]["volume_ratio"] = p.Volume
	params["audio"]["pitch_ratio"] = p.Pitch
	params["request"] = make(map[string]interface{})
	params["request"]["reqid"] = requestID
	params["request"]["text"] = text
	params["request"]["text_type"] = "plain"
	params["request"]["silence_duration"] = 300
	params["request"]["operation"] = opt
	params["request"]["with_frontend"] = "1"
	params["request"]["frontend_type"] = "unitTson"
	params["request"]["with_timestamp"] = "1"
	params["request"]["pure_english_opt"] = "1"
	resStr, _ := json.Marshal(params)
	return resStr
}

func (tts *TTS) parseResponse(res []byte) (resp SynResp, err error) {
	protoVersion := res[0] >> 4
	headSize := res[0] & 0x0f
	messageType := res[1] >> 4
	messageTypeSpecificFlags := res[1] & 0x0f
	serializationMethod := res[2] >> 4
	messageCompression := res[2] & 0x0f
	reserve := res[3]
	headerExtensions := res[4 : headSize*4]
	payload := res[headSize*4:]

	// log debug
	{
		log.Info(fmt.Sprintf("[bytedance_tts]Protocol version: %x - version %d",
			protoVersion, protoVersion))
		log.Info(fmt.Sprintf("[bytedance_tts]Header size: %x - %d bytes",
			headSize, headSize*4))
		log.Info(fmt.Sprintf("[bytedance_tts]Message type: %x - %s", messageType,
			enumMessageType[messageType]))
		log.Info(fmt.Sprintf("[bytedance_tts]Message type specific flags: %x - %s", messageTypeSpecificFlags,
			enumMessageTypeSpecificFlags[messageTypeSpecificFlags]))
		log.Info(fmt.Sprintf("[bytedance_tts]Message serialization method: %x - %s",
			serializationMethod, enumMessageSerializationMethods[serializationMethod]))
		log.Info(fmt.Sprintf("[bytedance_tts]Message compression: %x - %s",
			messageCompression, enumMessageCompression[messageCompression]))
		log.Info(fmt.Sprintf("[bytedance_tts]Reserved: %d", reserve))
	}

	if headSize != 1 {
		log.Warn(fmt.Sprintf("[bytedance_tts]Header extensions: %s",
			headerExtensions))
	}
	// audio-only server response
	if messageType == 0xb {
		// no sequence number as ACK
		if messageTypeSpecificFlags == 0 {
			log.Warn("[bytedance_tts]Payload size: 0")
		} else {
			sequenceNumber := int32(binary.BigEndian.Uint32(payload[0:4]))
			payloadSize := int32(binary.BigEndian.Uint32(payload[4:8]))
			payload = payload[8:]
			resp.Audio = append(resp.Audio, payload...)
			log.Info(fmt.Sprintf("[bytedance_tts]Sequence number: %d",
				sequenceNumber))
			log.Info(fmt.Sprintf("[bytedance_tts]Payload size: %d", payloadSize))

			if sequenceNumber < 0 {
				resp.IsLast = true
			}
		}
	} else if messageType == 0xf {
		code := int32(binary.BigEndian.Uint32(payload[0:4]))
		errMsg := payload[8:]
		if messageCompression == 1 {
			errMsg = gzipDecompress(errMsg)
		}
		log.Info(fmt.Sprintf("[bytedance_tts]Error code: %d", code))
		log.Info(fmt.Sprintf("[bytedance_tts]Error msg: %s", string(errMsg)))
		err = errors.New(string(errMsg))
		return
	} else if messageType == 0xc {

		payload = payload[4:]
		if messageCompression == 1 {
			payload = gzipDecompress(payload)
		}

		{
			info := &Addition{}

			if err = info.Unmarshal(payload); err != nil {
				return
			}

			resp.Addition = *info
		}
	} else {
		log.Error(fmt.Sprintf("wrong message type:%d", messageType))
		return
	}

	return
}

func (tts *TTS) getHeader() http.Header {
	return http.Header{"Authorization": []string{fmt.Sprintf("Bearer;%s", tts.config.Token)}}
}

func (tts *TTS) getWSSUrl() url.URL {
	return url.URL{Scheme: "wss", Host: tts.config.Addr, Path: path}
}
