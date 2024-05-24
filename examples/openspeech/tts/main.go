package main

import (
	"context"
	"fmt"
	"github.com/jyinz/volcano-sdk/openspeech/tts"

	uuid "github.com/satori/go.uuid"
)

func main() {
	cfg := tts.Config{
		AccessToken: "",
		AppID:       "",
	}

	cli := tts.New(cfg)
	err := cli.Synthesize(context.TODO(), tts.SynRequest{
		App: tts.APP{
			Token:   "access_token", // 任意非空字符串
			Cluster: "volcano_tts",
		},
		User: tts.User{
			Uid: "uid", // 任意非空字符串
		},
		Audio: tts.AudioConfig{
			VoiceType:       "BV700_streaming",
			Encoding:        "",
			CompressionRate: 0,
			Rate:            16000,
			SpeedRatio:      0,
			VolumeRatio:     0,
			PitchRatio:      0,
			Emotion:         "",
			Language:        "",
		},
		Request: tts.Request{
			Reqid:           uuid.NewV4().String(),
			Text:            "我想测试下语音合成的效果",
			TextType:        "",
			SilenceDuration: 300,
			WithFrontend:    0,
			FrontendType:    "",
			WithTimestamp:   1,
			SplitSentence:   1,
			PureEnglishOpt:  1,
		},
	}, func(result tts.SynResult) {
		fmt.Println("chunk size in response:", len(result.Chunk))
		fmt.Println("  frontend in response:", result.Frontend)
	})
	if err != nil {
		panic(err)
	}
}
