package repositories

import (
	"context"

	"medical-record-service/internal/domain/entities"

	"github.com/google/uuid"
)

type MedicalRecordRepositoryContract interface {
	Create(ctx context.Context, record *entities.MedicalRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error)
	Update(ctx context.Context, record *entities.MedicalRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error)
	ListAll(ctx context.Context) ([]*entities.MedicalRecord, error)
}
