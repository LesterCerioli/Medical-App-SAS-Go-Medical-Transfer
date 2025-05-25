package domain

import (
	"context"

	"github.com/google/uuid"
)

// PatientRepository defines the interface for patient data operations.
type PatientRepository interface {
	Create(ctx context.Context, patient *Patient) error
	GetByID(ctx context.Context, id uuid.UUID) (*Patient, error)
	Update(ctx context.Context, patient *Patient) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByEmail(ctx context.Context, email string) (*Patient, error)
	ListAll(ctx context.Context) ([]*Patient, error)
}
