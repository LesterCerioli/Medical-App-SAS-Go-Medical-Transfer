package services

import (
	"context"
	// Assuming your entities are now in a module path like:
	// "medical-record-service/internal/domain/entities"
	// Adjust if your module path is different.
	// Also, we might need a DTO for input instead of the entity directly.
	// For now, let's assume a placeholder DTO or a simplified entity.
	"medical-record-service/internal/domain/dtos" // Or a specific DTO for patient processing
)

// PatientData could be a specific DTO for incoming patient data
// For now, using CreatePatientRequest as a placeholder for data to be processed.
type PatientData dtos.CreatePatientRequest

// PatientServiceContract defines the operations for the patient processing service.
type PatientServiceContract interface {
	// Start begins the background processing of patient data.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the patient processing service.
	Stop(ctx context.Context) error
	// ProcessPatientData handles the processing of new or updated patient data.
	// This is a placeholder for what might be specific CRUD-like operations
	// or event-driven processing calls.
	ProcessPatientData(ctx context.Context, data PatientData) error

	// Example methods that could be derived from repository,
	// but adapted for service layer logic (e.g., with more business rules, logging, etc.)
	// GetPatientByID(ctx context.Context, id uuid.UUID) (*entities.Patient, error)
	// UpdatePatientInfo(ctx context.Context, patient *entities.Patient) error
}
