package openspeech

import "encoding/json"

type SynResp struct {
	Audio    []byte
	Addition Addition
	IsLast   bool
}

type Frontend struct {
	Words    []Word    `json:"words"`
	Phonemes []Phoneme `json:"phonemes"`
}

type Word struct {
	Word      string  `json:"word"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	UnitType  string  `json:"unit_type"`
}

type Phoneme struct {
	Phone     string  `json:"phone"`
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

type Addition struct {
	//Description string   `json:"description"`
	//Duration       string    `json:"duration"`
	//FirstPKG       string    `json:"first_pkg"`
	FrontendSTR    string    `json:"frontend,omitempty"` // 用于暂存
	FrontendStruct *Frontend `json:"frontendStruct"`
}

func (addition *Addition) Unmarshal(payload []byte) (err error) {
	if err = json.Unmarshal(payload, addition); err != nil {
		return
	}

	// get FrontendStruct
	if addition.FrontendSTR != "" {
		frontendStruct := &Frontend{}
		if err = json.Unmarshal([]byte(addition.FrontendSTR), frontendStruct); err != nil {
			return
		}

		addition.FrontendStruct = frontendStruct

		addition.FrontendSTR = ""
	}

	return
}
