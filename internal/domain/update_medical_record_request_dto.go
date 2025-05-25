package domain

import "encoding/json"

// UpdateMedicalRecordRequest defines the payload for updating an existing medical record.
type UpdateMedicalRecordRequest struct {
	RecordData json.RawMessage `json:"record_data,omitempty" validate:"omitempty"` // Allow partial update of record data
}
