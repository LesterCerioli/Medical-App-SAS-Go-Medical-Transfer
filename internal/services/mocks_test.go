package services

import (
	"context"
	// "encoding/json" // If any mock needs it (MockQueueAdapter will need it)
	"errors"      
	// "log"         // REMOVED: If any mock needs it (MockQueueAdapter will need it)
	// "sync"        // REMOVED: If any mock needs it (MockQueueAdapter will need it)
	"sync/atomic" 
	// "time"        // If any mock needs it

	// "medical-record-service/internal/adapters" // REMOVED: For QueueAdapter contract
	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"
	"github.com/google/uuid"
	// "github.com/stretchr/testify/mock" // Removed for now
)

// --- MockPatientRepository ---
// Compile-time check to ensure MockPatientRepository implements PatientRepositoryContract
var _ repositories.PatientRepositoryContract = (*MockPatientRepository)(nil)

// MockPatientRepository is a mock implementation of PatientRepositoryContract.
type MockPatientRepository struct {
	CreateFunc           func(ctx context.Context, patient *entities.Patient) error
	GetByIDFunc          func(ctx context.Context, id uuid.UUID) (*entities.Patient, error)
	UpdateFunc           func(ctx context.Context, patient *entities.Patient) error
	DeleteFunc           func(ctx context.Context, id uuid.UUID) error
	FindByEmailFunc      func(ctx context.Context, email string) (*entities.Patient, error)
	ListAllFunc          func(ctx context.Context) ([]*entities.Patient, error)
	
	ListAllFuncCallCount int32 
	CreateFuncCallCount  int32 
}

func (m *MockPatientRepository) Create(ctx context.Context, patient *entities.Patient) error {
	atomic.AddInt32(&m.CreateFuncCallCount, 1)
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, patient)
	}
	return nil
}

func (m *MockPatientRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Patient, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("GetByIDFunc not implemented in mock")
}

func (m *MockPatientRepository) Update(ctx context.Context, patient *entities.Patient) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, patient)
	}
	return errors.New("UpdateFunc not implemented in mock")
}

func (m *MockPatientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errors.New("DeleteFunc not implemented in mock")
}

func (m *MockPatientRepository) FindByEmail(ctx context.Context, email string) (*entities.Patient, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	return nil, errors.New("FindByEmailFunc not implemented in mock")
}

func (m *MockPatientRepository) ListAll(ctx context.Context) ([]*entities.Patient, error) {
	atomic.AddInt32(&m.ListAllFuncCallCount, 1)
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx) 
	}
	return nil, nil
}

// Other mock definitions will be placed here later.
