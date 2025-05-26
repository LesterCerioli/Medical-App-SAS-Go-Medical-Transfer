package dtos

import "github.com/google/uuid"

type TransferResponse struct {
	TransferID     uuid.UUID          `json:"transfer_id"`
	Status         string             `json:"status"` // e.g., "pending", "completed", "failed"
	Message        string             `json:"message,omitempty"`
	Patient        PatientDTO         `json:"patient"`
	MedicalRecords []MedicalRecordDTO `json:"medical_records"`
}
