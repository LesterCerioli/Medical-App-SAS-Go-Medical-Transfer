package dtos

import (
	"encoding/json"

	"github.com/google/uuid"
)

// CreateMedicalRecordRequest defines the payload for creating a new medical record.
type CreateMedicalRecordRequest struct {
	PatientID  uuid.UUID       `json:"patient_id" validate:"required"`
	RecordData json.RawMessage `json:"record_data" validate:"required"` // Expecting a valid JSON object
}
