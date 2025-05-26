package services

import (
	"context"
	// Assuming your DTOs are now in a module path like:
	// "medical-record-service/internal/domain/dtos"
	"medical-record-service/internal/domain/dtos" // Or a specific DTO for medical record processing
)

type MedicalRecordData dtos.CreateMedicalRecordRequest

type MedicalRecordServiceContract interface {
	Start(ctx context.Context) error

	Stop(ctx context.Context) error

	ProcessMedicalRecordData(ctx context.Context, data MedicalRecordData) error
}
