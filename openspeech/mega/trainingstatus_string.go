// Code generated by "stringer -type TrainingStatus"; DO NOT EDIT.

package mega

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TrainingStatusNotFound-0]
	_ = x[TrainingStatusTraining-1]
	_ = x[TrainingStatusSuccess-2]
	_ = x[TrainingStatusFailed-3]
	_ = x[TrainingStatusActive-4]
}

const _TrainingStatus_name = "TrainingStatusNotFoundTrainingStatusTrainingTrainingStatusSuccessTrainingStatusFailedTrainingStatusActive"

var _TrainingStatus_index = [...]uint8{0, 22, 44, 65, 85, 105}

func (i TrainingStatus) String() string {
	if i < 0 || i >= TrainingStatus(len(_TrainingStatus_index)-1) {
		return "TrainingStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TrainingStatus_name[_TrainingStatus_index[i]:_TrainingStatus_index[i+1]]
}
