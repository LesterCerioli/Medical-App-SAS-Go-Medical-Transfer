package services

import (
	"context"
	// Assuming your DTOs are now in a module path like:
	// "medical-record-service/internal/domain/dtos"
	"medical-record-service/internal/domain/dtos" // Or a specific DTO for medical record processing
)

// MedicalRecordData could be a specific DTO for incoming medical record data.
// For now, using CreateMedicalRecordRequest as a placeholder.
type MedicalRecordData dtos.CreateMedicalRecordRequest

// MedicalRecordServiceContract defines the operations for the medical record processing service.
type MedicalRecordServiceContract interface {
	// Start begins the background processing of medical records.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the medical record processing service.
	Stop(ctx context.Context) error
	// ProcessMedicalRecordData handles the processing of new or updated medical record data.
	ProcessMedicalRecordData(ctx context.Context, data MedicalRecordData) error

	// Example methods that could be derived from repository:
	// GetMedicalRecordByID(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error)
	// GetMedicalRecordsByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error)
}
