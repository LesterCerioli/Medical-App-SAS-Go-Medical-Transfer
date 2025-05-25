package repositories

import (
	"context"

	"github.com/google/uuid"
	"medical-record-service/internal/domain/entities"
)

// PatientRepository defines the interface for patient data operations.
type PatientRepository interface {
	Create(ctx context.Context, patient *entities.Patient) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Patient, error)
	Update(ctx context.Context, patient *entities.Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByEmail(ctx context.Context, email string) (*entities.Patient, error)
	ListAll(ctx context.Context) ([]*entities.Patient, error)
}
