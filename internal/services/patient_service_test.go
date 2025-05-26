package services

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	// Assuming entities and dtos are correctly pathed if needed by mocks/tests
	"medical-record-service/internal/domain/entities" 
	"medical-record-service/internal/domain/repositories" // For the contract
	"github.com/google/uuid"
)

// Compile-time check to ensure MockPatientRepository implements PatientRepositoryContract
var _ repositories.PatientRepositoryContract = (*MockPatientRepository)(nil)

// MockPatientRepository is a mock implementation of PatientRepositoryContract.
type MockPatientRepository struct {
	CreateFunc      func(ctx context.Context, patient *entities.Patient) error
	GetByIDFunc     func(ctx context.Context, id uuid.UUID) (*entities.Patient, error)
	UpdateFunc      func(ctx context.Context, patient *entities.Patient) error
	DeleteFunc      func(ctx context.Context, id uuid.UUID) error
	FindByEmailFunc func(ctx context.Context, email string) (*entities.Patient, error)
	ListAllFunc     func(ctx context.Context) ([]*entities.Patient, error)
}

func (m *MockPatientRepository) Create(ctx context.Context, patient *entities.Patient) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, patient)
	}
	return nil
}

func (m *MockPatientRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Patient, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockPatientRepository) Update(ctx context.Context, patient *entities.Patient) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, patient)
	}
	return nil
}

func (m *MockPatientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockPatientRepository) FindByEmail(ctx context.Context, email string) (*entities.Patient, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockPatientRepository) ListAll(ctx context.Context) ([]*entities.Patient, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
	}
	return nil, nil
}

// TestNewPatientService checks if the service is created.
func TestNewPatientService(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-patient-service: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger)

	if svc == nil {
		t.Errorf("NewPatientService() returned nil")
	}
}

// TestPatientService_StartStop tests the Start and Stop methods.
func TestPatientService_StartStop(t *testing.T) {
	mockRepo := &MockPatientRepository{
		ListAllFunc: func(ctx context.Context) ([]*entities.Patient, error) {
			// Called during shutdown path in Start's goroutine for demo
			t.Log("MockPatientRepository.ListAll called during shutdown simulation")
			return nil, nil
		},
	}
	logger := log.New(os.Stdout, "test-startstop: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger).(*PatientServiceImpl) // Cast to access stopChan if needed, or just test behavior

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give some time for the goroutine to run (e.g., one tick)
	time.Sleep(100 * time.Millisecond) // Shorter than ticker for quick test

	err = svc.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Check if stopChan is closed (indirectly, goroutine should exit)
	// We can wait a bit and see if the logger prints "stopping" messages.
	// For more robust check, would need to inspect internal state or use waitgroups.
	time.Sleep(100 * time.Millisecond) 
}

// TestPatientService_ProcessPatientData tests the ProcessPatientData method.
func TestPatientService_ProcessPatientData(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	// Potentially set CreateFunc on mockRepo if ProcessPatientData actually calls it.
	// For now, the service's ProcessPatientData is a placeholder.

	logger := log.New(os.Stdout, "test-processdata: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger)

	testData := PatientData{ // Using PatientData alias for dtos.CreatePatientRequest
		Name:        "Test User",
		DateOfBirth: "2000-01-01",
		Email:       "test@example.com",
	}

	err := svc.ProcessPatientData(context.Background(), testData)
	if err != nil {
		t.Errorf("ProcessPatientData() error = %v", err)
	}
	// Add assertions here:
	// - Check if logger was called (would need a mock logger or log output capturing).
	// - If ProcessPatientData called repo methods, check if mockRepo functions were called.
}

// TestPatientService_Start_ContextCancellation tests if service stops on context cancellation.
func TestPatientService_Start_ContextCancellation(t *testing.T) {
	listAllCalled := false
	mockRepo := &MockPatientRepository{
		ListAllFunc: func(ctx context.Context) ([]*entities.Patient, error) {
			listAllCalled = true
			t.Log("MockPatientRepository.ListAll called due to context cancellation")
			return nil, nil
		},
	}
	logger := log.New(os.Stdout, "test-ctxcancel: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger).(*PatientServiceImpl)

	ctx, cancel := context.WithCancel(context.Background())

	err := svc.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context to trigger shutdown path
	cancel()

	// Wait for goroutine to potentially exit and call ListAll
	// This timeout should be long enough for the select case to be hit
	time.Sleep(200 * time.Millisecond) 

	if !listAllCalled {
		t.Errorf("Expected ListAll to be called on context cancellation, but it wasn't")
	}
	
	// Ensure Stop can still be called without issues
	err = svc.Stop(context.Background()) // Use a new context for Stop
	if err != nil {
		t.Fatalf("Stop() after context cancellation error = %v", err)
	}
}
