package services

import (
	"context"
	"encoding/json"
	"errors" // For errors.New
	"log"
	"os"
	"sync/atomic" 
	"testing"
	"time"
	"fmt" // For fmt.Sprintf in logging

	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"
	// No direct import of dtos needed here if MedicalRecordData is used from the same 'services' package
	// "medical-record-service/internal/domain/dtos" 
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/mock" // Not using testify/mock features for these mocks
)

// Compile-time check to ensure MockMedicalRecordRepository implements MedicalRecordRepositoryContract
var _ repositories.MedicalRecordRepositoryContract = (*MockMedicalRecordRepository)(nil)

// MockMedicalRecordRepository is a mock implementation of MedicalRecordRepositoryContract.
type MockMedicalRecordRepository struct {
	CreateFunc            func(ctx context.Context, record *entities.MedicalRecord) error
	GetByIDFunc           func(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error)
	UpdateFunc            func(ctx context.Context, record *entities.MedicalRecord) error
	DeleteFunc            func(ctx context.Context, id uuid.UUID) error
	FindByPatientIDFunc func(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error)
	ListAllFunc           func(ctx context.Context) ([]*entities.MedicalRecord, error)

	ListAllFuncCallCount int32 
	CreateFuncCallCount  int32 
}

func (m *MockMedicalRecordRepository) Create(ctx context.Context, record *entities.MedicalRecord) error {
	atomic.AddInt32(&m.CreateFuncCallCount, 1)
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, record)
	}
	return nil
}

func (m *MockMedicalRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("GetByIDFunc not implemented in mock")
}

func (m *MockMedicalRecordRepository) Update(ctx context.Context, record *entities.MedicalRecord) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, record)
	}
	return errors.New("UpdateFunc not implemented in mock")
}

func (m *MockMedicalRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errors.New("DeleteFunc not implemented in mock")
}

func (m *MockMedicalRecordRepository) FindByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error) {
	if m.FindByPatientIDFunc != nil {
		return m.FindByPatientIDFunc(ctx, patientID)
	}
	return nil, errors.New("FindByPatientIDFunc not implemented in mock")
}

func (m *MockMedicalRecordRepository) ListAll(ctx context.Context) ([]*entities.MedicalRecord, error) {
	atomic.AddInt32(&m.ListAllFuncCallCount, 1)
	// t.Logf("MockMedicalRecordRepository.ListAll called, count: %d", atomic.LoadInt32(&m.ListAllFuncCallCount)) // Debugging line
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
	}
	return nil, nil
}

func TestNewMedicalRecordService(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-service: ", log.LstdFlags)
	svc := NewMedicalRecordService(mockRepo, logger)
	assert.NotNil(t, svc, "NewMedicalRecordService() should not return nil")
}

func TestMedicalRecordService_WorkersProcessJobs_And_GracefulShutdown(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-process-shutdown: ", log.LstdFlags)
	
	// Explicitly cast to *MedicalRecordServiceImpl to access numWorkers
	svcImpl := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)
	assert.NotNil(t, svcImpl, "NewMedicalRecordService returned nil or not MedicalRecordServiceImpl")

	// Service's main operational context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures context is cancelled if test panics or exits early

	err := svcImpl.Start(ctx)
	assert.NoError(t, err, "Start() should not return an error")

	numJobs := 3 * svcImpl.numWorkers // Send enough jobs for all workers
	for i := 0; i < numJobs; i++ {
		jobData := MedicalRecordData{ 
			PatientID:  uuid.New(),
			RecordData: json.RawMessage(fmt.Sprintf(`{"detail":"record %d"}`, i)),
		}
		// Use a timeout for sending to the job channel to prevent test hanging
		sendCtx, sendCancel := context.WithTimeout(ctx, 200*time.Millisecond) // Increased timeout
		err := svcImpl.ProcessMedicalRecordData(sendCtx, jobData)
		sendCancel() // Always call cancel for contexts created with WithTimeout or WithCancel
		assert.NoError(t, err, "ProcessMedicalRecordData() should not error for job %d", i)
	}

	// Allow time for all jobs to be picked up and processed.
	// Calculation: numJobs / numWorkers * (processJob_time + small_buffer)
	// processJob_time is 100ms. If numWorkers is 5, (15/5)*100ms = 300ms.
	// Add a generous buffer.
	time.Sleep(time.Duration(numJobs/svcImpl.numWorkers+1)*150*time.Millisecond + 200*time.Millisecond)


	// Stop the service
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() should not return an error")

	// Verify that all jobs were processed
	// processJob in MedicalRecordServiceImpl calls ListAll once per job.
	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, int32(numJobs), processedCount, "Expected all jobs to be processed")

	// Try to send a job after Stop. It should fail because serviceCtx is cancelled.
	errAfterStop := svcImpl.ProcessMedicalRecordData(context.Background(), MedicalRecordData{PatientID: uuid.New()})
	assert.Error(t, errAfterStop, "ProcessMedicalRecordData after Stop should return an error")
	assert.EqualError(t, errAfterStop, "service is shutting down, cannot accept new medical record jobs", "Error message mismatch")
}

func TestMedicalRecordService_Start_ContextCancellation_StopsProcessing(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-ctxcancel: ", log.LstdFlags)
	svcImpl := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)

	serviceCtx, cancelServiceCtx := context.WithCancel(context.Background())

	err := svcImpl.Start(serviceCtx) // Start service with this cancellable context
	assert.NoError(t, err, "Start() should not return an error")

	// Send a few jobs
	numInitialJobs := svcImpl.numWorkers // Send one job per worker initially
	for i := 0; i < numInitialJobs; i++ {
		jobData := MedicalRecordData{PatientID: uuid.New(), RecordData: json.RawMessage(`{"detail":"initial job"}`)}
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_ = svcImpl.ProcessMedicalRecordData(sendCtx, jobData) // Ignore error for this test focus
		sendCancel()
	}

	// Allow some jobs to be processed
	time.Sleep(time.Duration(numInitialJobs/svcImpl.numWorkers+1) * 50 * time.Millisecond)


	cancelServiceCtx() // Cancel the service's main context

	// Wait for shutdown triggered by context cancellation to complete.
	// A simple way is to wait a bit, assuming shutdown is quick.
	// A more robust way would be to check a signal from a Stop-like mechanism or if jobChan is closed.
	// Since Stop() itself now signals a channel that shutdown() listens to,
	// and shutdown calls serviceCancel(), this test primarily checks that serviceCancel()
	// stops the workers and job processing.
	time.Sleep(200 * time.Millisecond) // Allow time for workers to react to context cancellation

	processedCountBeforeCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	t.Logf("Jobs processed before/during cancellation: %d", processedCountBeforeCancel)
	assert.True(t, processedCountBeforeCancel <= int32(numInitialJobs), "Should process at most the initial jobs")


	// Attempt to send new jobs; should fail because serviceCtx is cancelled.
	errAfterCtxCancel := svcImpl.ProcessMedicalRecordData(context.Background(), MedicalRecordData{PatientID: uuid.New()})
	assert.Error(t, errAfterCtxCancel, "ProcessMedicalRecordData after context cancellation should return an error")
	assert.EqualError(t, errAfterCtxCancel, "service is shutting down, cannot accept new medical record jobs", "Error message mismatch")

	// Ensure that ListAllFuncCallCount does not increase further after cancellation.
	time.Sleep(200 * time.Millisecond) // Wait to see if any more jobs are processed
	processedCountAfterCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, processedCountBeforeCancel, processedCountAfterCancel, "No new jobs should be processed after context cancellation")

	// Call Stop for completeness, it should be idempotent or quick if already stopped.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() after context cancellation should not error")
}
