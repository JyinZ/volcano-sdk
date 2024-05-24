package sami

const (
	EventStartTask    = "StartTask"
	EventTaskStarted  = "TaskStarted"
	EventFinishTask   = "FinishTask"
	EventTaskFinished = "TaskFinished"
	EventTaskRequest  = "TaskRequest"
)

type (
	// WebSocketRequest websocket请求体包
	WebSocketRequest struct {
		Token     string `header:"SAMI-Token,required" json:"token,required" query:"token,required"`
		Appkey    string `json:"appkey,required" query:"appkey,required" vd:"$!=''"`
		Namespace string `json:"namespace,required" query:"namespace,required" vd:"$!=''"`
		Version   string `json:"version,omitempty" query:"version"`
		Event     string `json:"event,omitempty" query:"event"`
		Payload   string `form:"payload" json:"payload,omitempty"`
		Data      []byte `form:"data" json:"data,omitempty"`
		TaskId    string `json:"task_id,omitempty" query:"task_id"`
	}

	// WebSocketResponse websocket返回包
	WebSocketResponse struct {
		TaskId     string `form:"task_id,required" json:"task_id,required" query:"task_id,required"`
		MessageId  string `form:"message_id,required" json:"message_id,required" query:"message_id,required"`
		Namespace  string `form:"namespace,required" json:"namespace,required" query:"namespace,required"`
		Event      string `form:"event,required" json:"event,required" query:"event,required"`
		StatusCode int32  `form:"status_code,required" json:"status_code,required" query:"status_code,required"`
		StatusText string `form:"status_text,required" json:"status_text,required" query:"status_text,required"`
		Payload    string `form:"payload,omitempty" json:"payload,omitempty" query:"payload,omitempty"`
		Data       []byte `form:"data,omitempty" json:"data,omitempty" query:"data,omitempty"`
	}
)

func (rsp WebSocketResponse) Started() bool {
	return rsp.Event == EventTaskStarted
}

func (rsp WebSocketResponse) Finished() bool {
	return rsp.Event == EventTaskFinished
}
