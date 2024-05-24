package tts

import "fmt"

type Error struct {
	Code int32
	Msg  string
}

func (e Error) Error() string {
	return fmt.Sprintf("code=%d, desc=%s", e.Code, e.Msg)
}
