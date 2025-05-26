package dtos

import "encoding/json"

type UpdateMedicalRecordRequest struct {
	RecordData json.RawMessage `json:"record_data,omitempty" validate:"omitempty"` // Allow partial update of record data
}
