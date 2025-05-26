package services

import (
	"context"
	"errors" 
	"fmt"    
	"log"
	"os"
	"sync/atomic" 
	"testing"
	"time"

	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"
	// PatientData is defined in patient_service_contract.go (same package)
	// and is an alias for dtos.CreatePatientRequest.
	// The contract file imports dtos, so the type is available.
	// "medical-record-service/internal/domain/dtos" 
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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

func TestNewPatientService(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-patient-service: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger)
	assert.NotNil(t, svc, "NewPatientService() should not return nil")
}

func TestPatientService_WorkersProcessJobs_And_GracefulShutdown(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-ps-process-shutdown: ", log.LstdFlags)
	
	svcImpl := NewPatientService(mockRepo, logger).(*PatientServiceImpl) 
	assert.NotNil(t, svcImpl, "NewPatientService returned nil or not PatientServiceImpl")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() 

	err := svcImpl.Start(ctx)
	assert.NoError(t, err, "Start() should not return an error")

	numJobs := 3 * svcImpl.numWorkers 
	for i := 0; i < numJobs; i++ {
		jobData := PatientData{
			Name:        fmt.Sprintf("Test User %d", i), 
			Email:       fmt.Sprintf("test%d@example.com", i), 
			DateOfBirth: "2000-01-01",
		}
		sendCtx, sendCancel := context.WithTimeout(ctx, 200*time.Millisecond) 
		err := svcImpl.ProcessPatientData(sendCtx, jobData)
		sendCancel() 
		assert.NoError(t, err, "ProcessPatientData() should not error for job %d", i)
	}
	
	// Allow time for jobs to be processed.
	// Calculation: (numJobs / numWorkers) * (processJob_time (100ms) + small_buffer)
	// Example: (15 jobs / 5 workers) * 100ms = 300ms. Add generous buffer.
	time.Sleep(time.Duration(numJobs/svcImpl.numWorkers+1)*150*time.Millisecond + 200*time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() should not return an error")

	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, int32(numJobs), processedCount, "Expected all jobs to be processed")

	errAfterStop := svcImpl.ProcessPatientData(context.Background(), PatientData{Name: "After Stop"})
	assert.Error(t, errAfterStop, "ProcessPatientData after Stop should return an error")
	assert.EqualError(t, errAfterStop, "patient service is shutting down, cannot accept new jobs", "Error message mismatch")
}

func TestPatientService_Start_ContextCancellation_StopsProcessing(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-ps-ctxcancel: ", log.LstdFlags)
	svcImpl := NewPatientService(mockRepo, logger).(*PatientServiceImpl)

	serviceCtx, cancelServiceCtx := context.WithCancel(context.Background())

	err := svcImpl.Start(serviceCtx) 
	assert.NoError(t, err, "Start() should not return an error")

	numInitialJobs := svcImpl.numWorkers 
	for i := 0; i < numInitialJobs; i++ {
		jobData := PatientData{Name: fmt.Sprintf("Initial Job %d", i), Email: fmt.Sprintf("initial%d@example.com", i), DateOfBirth: "2000-01-01"}
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_ = svcImpl.ProcessPatientData(sendCtx, jobData) 
		sendCancel()
	}

	time.Sleep(time.Duration(numInitialJobs/svcImpl.numWorkers+1) * 50 * time.Millisecond)

	cancelServiceCtx() // Cancel the service's main context

	time.Sleep(200 * time.Millisecond) // Allow time for workers to react

	processedCountBeforeCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	t.Logf("Jobs processed by PatientService before/during cancellation: %d", processedCountBeforeCancel)
	assert.True(t, processedCountBeforeCancel <= int32(numInitialJobs), "Should process at most the initial jobs")

	errAfterCtxCancel := svcImpl.ProcessPatientData(context.Background(), PatientData{Name: "Post Cancel Job"})
	assert.Error(t, errAfterCtxCancel, "ProcessPatientData after context cancellation should return an error")
	assert.EqualError(t, errAfterCtxCancel, "patient service is shutting down, cannot accept new jobs", "Error message mismatch")

	time.Sleep(200 * time.Millisecond) 
	processedCountAfterCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, processedCountBeforeCancel, processedCountAfterCancel, "No new jobs should be processed after context cancellation")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() after context cancellation should not error")
}
