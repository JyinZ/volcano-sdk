package sami

type WebSocketRequest struct {
	Token     string  `header:"SAMI-Token,required" json:"token,required" query:"token,required"`
	Appkey    string  `json:"appkey,required" query:"appkey,required" vd:"$!=''"`
	Namespace string  `json:"namespace,required" query:"namespace,required" vd:"$!=''"`
	Version   string  `json:"version,omitempty" query:"version"`
	Event     string  `json:"event,omitempty" query:"event"`
	Payload   *string `form:"payload" json:"payload,omitempty"`
	Data      []byte  `form:"data" json:"data,omitempty"`
	TaskId    string  `json:"task_id,omitempty" query:"task_id"`
}

type VoiceConversionRequest struct {
	Info        *AudioInfo   `json:"audio_info,omitempty"`
	AudioConfig *AudioConfig `json:"audio_config,omitempty"`
	Speaker     *string      `json:"speaker,omitempty"`
}

type AudioInfo struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Channel    int    `json:"channel,omitempty"`
	Format     string `json:"format,omitempty"`
}

type AudioConfig struct {
	Format     string `json:"format,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"` // default 24000, [8000, 16000, 22050, 24000, 32000, 44100, 48000]
	Channel    int    `json:"channel,omitempty"`     // 1, 2
}
