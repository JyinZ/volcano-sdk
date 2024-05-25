package main

import (
	"context"
	"fmt"
	"github.com/jyinz/volcano-sdk"
	"github.com/jyinz/volcano-sdk/sami/vc"
	"time"
)

func main() {
	cfg := vc.Config{
		Config: volcano.Config{
			AccessKey: "",
			SecretKey: "",
		},
		AppKey: "",
	}

	audio := make(chan []byte, 64)
	go func() {
		defer close(audio)

		// 16k单声道，每秒25帧速率发送,每帧1280字节
		tcr := time.NewTicker(40 * time.Millisecond)
		defer tcr.Stop()

		for i := 0; i < 1000; i++ {
			audio <- make([]byte, 1280)
			<-tcr.C
		}
	}()

	cli := vc.New(cfg)
	err := cli.Conversion(context.TODO(), vc.VoiceConversionRequest{
		Speaker: "zh_female_tianmei_stream",
		AudioInfo: vc.AudioInfo{
			SampleRate: 16000,
			Channel:    1,
			Format:     "s16le",
		},
		AudioConfig: vc.AudioInfo{
			SampleRate: 16000,
			Channel:    1,
			Format:     "s16le",
		},
		Extra: &vc.Extra{
			DownstreamAlign: true,
		},
	},
		audio,
		func(chunk []byte) {
			fmt.Println("chunk size:", len(chunk))
		})
	if err != nil {
		panic(err)
	}

}
