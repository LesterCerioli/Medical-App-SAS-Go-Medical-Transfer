package dtos

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateMedicalRecordRequest struct {
	PatientID  uuid.UUID       `json:"patient_id" validate:"required"`
	RecordData json.RawMessage `json:"record_data" validate:"required"` // Expecting a valid JSON object
}
