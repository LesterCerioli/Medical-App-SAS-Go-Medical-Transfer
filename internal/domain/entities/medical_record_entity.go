package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MedicalRecord represents a medical record for a patient.
// The RecordData field is stored as JSONB in the database.
type MedicalRecord struct {
	ID         uuid.UUID       `json:"id" db:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	PatientID  uuid.UUID       `json:"patient_id" db:"patient_id" gorm:"type:uuid;not null"` // foreignKey:PatientID is implicitly handled by GORM if the field name is PatientID and Patient struct has an ID. Explicit tagging can be added if needed.
	RecordData json.RawMessage `json:"record_data" db:"record_data" gorm:"type:jsonb;not null"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at" gorm:"not null"` // gorm will default to autoCreateTime
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at" gorm:"not null"` // gorm will default to autoUpdateTime
}
