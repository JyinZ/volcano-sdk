package sami

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math"
	"net/http"
	"net/url"
	"time"

	log "digital-robot3d/pkg/log"
)

const (
	EventStartTask    = "StartTask"
	EventTaskStarted  = "TaskStarted"
	EventFinishTask   = "FinishTask"
	EventTaskFinished = "TaskFinished"
	EventTaskRequest  = "TaskRequest"

	outputSampleRate = 16000
	outputChannel    = 1
	Format           = "s16le"
	step             = 3200

	namespace = "VoiceConversionStream"

	LogFlag  = "[bytedance_vc]"
	sendrecv = "[send_recv]"
)

type VoiceConversion struct {
	msgBuff chan []byte

	c *websocket.Conn

	addr, speaker       string
	appKey, accessToken string

	closeChan chan int

	request *WebSocketRequest
}

func NewVoiceConversion(addr, speaker string, accessKey, secretKey, appKey string) (*VoiceConversion, error) {
	log.Info(LogFlag + "init vc...")

	tkn, err := GetToken(accessKey, secretKey, appKey)
	if err != nil {
		return nil, err
	}

	vc := &VoiceConversion{
		msgBuff:     make(chan []byte, 1024),
		addr:        addr,
		speaker:     speaker,
		appKey:      appKey,
		accessToken: tkn,
		closeChan:   make(chan int),
	}

	// 建连
	if err = vc.init(); err != nil {
		return nil, err
	}

	// 启动任务
	if err = vc.StartEvent(); err != nil {
		return nil, err
	}

	// 发送跳
	go vc.ping()

	return vc, nil
}

func (vc *VoiceConversion) StartEvent() (err error) {
	testSpeaker := vc.speaker
	voiceConversionPayload := VoiceConversionRequest{
		Speaker: &testSpeaker,
		Info: &AudioInfo{
			SampleRate: 16000,
			Channel:    1,
			Format:     Format,
		},
		AudioConfig: &AudioConfig{
			Format:     Format,
			Channel:    1,
			SampleRate: 16000,
		},
	}

	vc.request = &WebSocketRequest{
		Token:     vc.accessToken,
		Appkey:    vc.appKey,
		Namespace: namespace,
		Event:     EventStartTask,
	}
	b, _ := json.Marshal(&voiceConversionPayload)
	plStr := string(b)
	vc.request.Payload = &plStr
	fmt.Println("req payload: ", *vc.request.Payload)

	controlMsg, _ := json.Marshal(vc.request)
	_ = vc.c.WriteMessage(websocket.TextMessage, controlMsg)
	err = vc.readTaskStartedEvent()

	return
}

func (vc *VoiceConversion) Close() {
	close(vc.closeChan)
	close(vc.msgBuff)
	vc.c.Close()

	log.Info(LogFlag + "close connect")
}

func (vc *VoiceConversion) FinishTask() error {
	vc.request.Event = EventFinishTask
	vc.request.Payload = nil
	controlMsg, _ := json.Marshal(vc.request)
	return vc.c.WriteMessage(websocket.TextMessage, controlMsg)
}

func (vc *VoiceConversion) SendPCMToChan(data []byte) {
	vc.msgBuff <- data
}

func (vc *VoiceConversion) Sender() error {
	for data := range vc.msgBuff {
		l := len(data)

		log.Info(LogFlag+"coming in sender.", "data length is ", l)

		if l < 1 {
			return nil
		}

		times := int(math.Ceil(float64(l) / float64(step)))

		for i := 0; i < times; i++ {
			var dataToSend []byte
			if (i+1)*step > len(data) {
				dataToSend = data[i*step:]
			} else {
				dataToSend = data[i*step : (i+1)*step]
			}

			if len(dataToSend) > 0 {
				log.Info(LogFlag, sendrecv, fmt.Sprintf("ready to send data length[%d]", len(dataToSend)))

				err := vc.c.WriteMessage(websocket.BinaryMessage, dataToSend)
				if err != nil {
					log.Error(LogFlag, "send data to vc engine error", err.Error())
					return err
				}
			}
			log.Info(LogFlag, sendrecv, fmt.Sprintf("sended len[%v]", len(dataToSend)))
		}
	}

	return nil
}

func (vc *VoiceConversion) RECV(callback func(msg []byte)) error {

	log.Info(LogFlag + "coming in receiver...")
LOOP:
	for {
		select {
		case <-vc.closeChan:
			return nil
		default:
			mt, message, err := vc.c.ReadMessage()
			if err != nil {
				log.Error(LogFlag, "read error", err.Error())
				return err
			}
			if mt == websocket.BinaryMessage {
				log.Info(LogFlag, sendrecv, fmt.Sprintf("binary, recv: byte[%v]", len(message)))
				callback(message)
			} else {
				log.Info(LogFlag, sendrecv, fmt.Sprintf("text, recv: byte[%v]", len(message)))

				wsResp := WebSocketResponse{}
				err = json.Unmarshal(message, &wsResp)
				if err != nil {
					log.Error(LogFlag, "recv text message, parse failed", err.Error())
					log.Debug(string(message))
					continue
				}

				if wsResp.Event == EventTaskFinished {
					log.Info(LogFlag + "event finish")
					break LOOP
				}
				if wsResp.Payload != nil {
					log.Debug(LogFlag+*wsResp.Payload, time.Now())
				}
				callback(wsResp.Data)
			}
		}
	}

	return nil
}

func (vc *VoiceConversion) readTaskStartedEvent() error {
	msgType, message, err := vc.c.ReadMessage()
	if err != nil {
		log.Error(LogFlag, "read TaskStarted event failed", err)
		return err
	}
	if msgType != websocket.TextMessage {
		log.Error(LogFlag, "read TaskStarted event failed, message type not TextMessage", msgType)
		return fmt.Errorf("MessageTypeNotMatch")
	}

	taskStartedEvent := &WebSocketResponse{}
	err = json.Unmarshal(message, taskStartedEvent)
	if err != nil {
		log.Error(LogFlag, "Unmarshal failed", err)
		return err
	}
	if taskStartedEvent.Event != EventTaskStarted {
		log.Error(LogFlag, "read TaskStarted event failed, event type not match", *taskStartedEvent)
		return fmt.Errorf("EventTypeNotMatch")
	}

	b, _ := json.Marshal(taskStartedEvent)
	log.Debug(LogFlag, ":", string(b))
	return nil
}

func (vc *VoiceConversion) init() (err error) {
	u := url.URL{Scheme: "wss", Host: vc.addr, Path: "/api/v1/ws"}
	requestHeader := http.Header{}

	vc.c, _, err = websocket.DefaultDialer.Dial(u.String(), requestHeader)

	return err
}

func (vc *VoiceConversion) ping() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := vc.c.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(11*time.Second)); err != nil {
				fmt.Printf("ping error: %v\n", err)
				return
			}

			// ping 通，即可设置write deadline
			_ = vc.c.SetWriteDeadline(time.Now().Add(20 * time.Second))
		case <-vc.closeChan:
			return
		}
	}

}
