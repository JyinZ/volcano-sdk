package mega

import (
	"errors"
	"fmt"
)

type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("code=%d, desc=%s", e.Code, e.Msg)
}

func (e *Error) Message() string {
	return msgs[e.Code][1]
}

var msgs = map[int][2]string{
	1001: {"BadRequestError", "请求参数有误"},
	1101: {"AudioUploadError", "音频上传失败"},
	1102: {"ASRError", "ASR（语音识别成文字）转写失败"},
	1103: {"SIDError", "SID声纹检测失败"},
	1104: {"SIDFailError", "声纹检测未通过，声纹跟名人相似度过高"},
	1105: {"GetAudioDataError", "获取音频数据失败"},
	1106: {"SpeakerIDDuplicationError", "SpeakerID重复"},
	1107: {"SpeakerIDNotFoundError", "SpeakerID未找到"},
	1108: {"AudioConvertError", "音频转码失败"},
	1109: {"WERError", "wer检测错误，上传音频与请求携带文本对比字错率过高"},
	1111: {"AEDError", "aed检测错误，通常由于音频不包含说话声"},
	1112: {"SNRError", "SNR检测错误，通常由于信噪比过高"},
	1113: {"DenoiseError", "降噪处理失败"},
	1114: {"AudioQualityError", "音频质量低，降噪失败"},
	1122: {"ASRNoSpeakerError", "未检测到人声"},
	1123: {"MaxUpload", "上传接口已经达到次数限制，目前同一个音色支持10次上传"},
}

func NewError(code int) error {
	return &Error{
		Code: code,
		Msg:  msgs[code][0],
	}
}

func Translate(err error) string {
	var e = new(Error)
	errors.As(err, e)
	return e.Message()
}
