package repositories

import (
	"context"

	"medical-record-service/internal/domain/entities"

	"github.com/google/uuid"
)

type PatientRepositoryContract interface {
	Create(ctx context.Context, patient *entities.Patient) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Patient, error)
	Update(ctx context.Context, patient *entities.Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByEmail(ctx context.Context, email string) (*entities.Patient, error)
	ListAll(ctx context.Context) ([]*entities.Patient, error)
}
