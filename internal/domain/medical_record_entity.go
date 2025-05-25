package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MedicalRecord represents a medical record for a patient.
// The RecordData field is stored as JSONB in the database.
type MedicalRecord struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	PatientID  uuid.UUID       `json:"patient_id" db:"patient_id"`
	RecordData json.RawMessage `json:"record_data" db:"record_data"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}
