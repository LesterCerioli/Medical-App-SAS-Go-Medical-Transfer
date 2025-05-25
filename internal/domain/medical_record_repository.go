package domain

import (
	"context"

	"github.com/google/uuid"
)

// MedicalRecordRepository defines the interface for medical record data operations.
type MedicalRecordRepository interface {
	Create(ctx context.Context, record *MedicalRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*MedicalRecord, error)
	Update(ctx context.Context, record *MedicalRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByPatientID(ctx context.Context, patientID uuid.UUID) ([]*MedicalRecord, error)
	ListAll(ctx context.Context) ([]*MedicalRecord, error)
}
