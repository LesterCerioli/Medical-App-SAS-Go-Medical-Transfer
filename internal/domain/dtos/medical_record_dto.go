package dtos

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MedicalRecordDTO represents medical record data in API responses.
type MedicalRecordDTO struct {
	ID         uuid.UUID       `json:"id"`
	PatientID  uuid.UUID       `json:"patient_id"`
	RecordData json.RawMessage `json:"record_data"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
