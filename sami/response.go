package sami

type WebSocketResponse struct {
	TaskId     string  `form:"task_id,required" json:"task_id,required" query:"task_id,required"`
	MessageId  string  `form:"message_id,required" json:"message_id,required" query:"message_id,required"`
	Namespace  string  `form:"namespace,required" json:"namespace,required" query:"namespace,required"`
	Event      string  `form:"event,required" json:"event,required" query:"event,required"`
	StatusCode int32   `form:"status_code,required" json:"status_code,required" query:"status_code,required"`
	StatusText string  `form:"status_text,required" json:"status_text,required" query:"status_text,required"`
	Payload    *string `form:"payload,omitempty" json:"payload,omitempty" query:"payload,omitempty"`
	Data       []byte  `form:"data,omitempty" json:"data,omitempty" query:"data,omitempty"`
}

type GetTokenResponse struct {
	StatusCode int32  `form:"status_code,required" json:"status_code,required" query:"status_code,required"`
	StatusText string `form:"status_text,required" json:"status_text,required" query:"status_text,required"`
	TaskId     string `form:"task_id,required" json:"task_id,required" query:"task_id,required"`
	Token      string `form:"token,required" json:"token,required" query:"token,required"`
	ExpiresAt  int64  `form:"expires_at,required" json:"expires_at,required" query:"expires_at,required"`
}
