package repositories

import (
	"context"

	"github.com/google/uuid"
	"medical-record-service/internal/domain/entities"
)

// MedicalRecordRepository defines the interface for medical record data operations.
type MedicalRecordRepository interface {
	Create(ctx context.Context, record *entities.MedicalRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error)
	Update(ctx context.Context, record *entities.MedicalRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error)
	ListAll(ctx context.Context) ([]*entities.MedicalRecord, error)
}
