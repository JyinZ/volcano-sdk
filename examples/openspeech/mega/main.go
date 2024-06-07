package main

import (
	"context"
	"fmt"
	"github.com/jyinz/volcano-sdk"
	"github.com/jyinz/volcano-sdk/openspeech/mega"
)

func main() {
	cfg := mega.Config{
		Config: volcano.Config{
			AccessKey: "",
			SecretKey: "",
		},
		AccessToken: "",
		AppID:       "",
	}

	cli := mega.New(cfg)
	ls, err := cli.ListSpeakers(context.TODO(), &mega.BatchListMegaTTSTrainStatusRequest{
		LiseMegaTTSTrainStatusRequest: mega.LiseMegaTTSTrainStatusRequest{
			SpeakerIDs: nil,
		},
		PageNumber: 0,
		PageSize:   0,
		NextToken:  "",
		MaxResults: 0,
	})
	if err != nil {
		panic(err)
	}

	for _, status := range ls.Statuses {
		fmt.Println("speaker:", fmt.Sprintf("%+v", status))
	}
}
