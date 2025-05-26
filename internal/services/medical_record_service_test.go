package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync/atomic" // For atomic counters
	"testing"
	"time"

	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"
	"github.com/google/uuid"
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

	ListAllFuncCallCount int32 // Atomic counter
	CreateFuncCallCount  int32 // Atomic counter
}

func (m *MockMedicalRecordRepository) Create(ctx context.Context, record *entities.MedicalRecord) error {
	atomic.AddInt32(&m.CreateFuncCallCount, 1)
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, record)
	}
	return nil
}

func (m *MockMedicalRecordRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error) {
	// Implement if needed for tests, or leave as placeholder
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockMedicalRecordRepository) Update(ctx context.Context, record *entities.MedicalRecord) error {
	// Implement if needed for tests
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, record)
	}
	return nil
}

func (m *MockMedicalRecordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Implement if needed for tests
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockMedicalRecordRepository) FindByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error) {
	// Implement if needed for tests
	if m.FindByPatientIDFunc != nil {
		return m.FindByPatientIDFunc(ctx, patientID)
	}
	return nil, nil
}

func (m *MockMedicalRecordRepository) ListAll(ctx context.Context) ([]*entities.MedicalRecord, error) {
	atomic.AddInt32(&m.ListAllFuncCallCount, 1)
	if m.ListAllFunc != nil {
		return m.ListAllFunc(ctx)
	}
	return nil, nil
}

// TestNewMedicalRecordService (can remain the same)
func TestNewMedicalRecordService(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-service: ", log.LstdFlags)
	svc := NewMedicalRecordService(mockRepo, logger)
	if svc == nil {
		t.Errorf("NewMedicalRecordService() returned nil")
	}
}

// TestMedicalRecordService_ProcessJobsAndShutdown
func TestMedicalRecordService_ProcessJobsAndShutdown(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-process-shutdown: ", log.LstdFlags)
	svc := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := svc.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	numJobs := 8 // A different number to differentiate from PatientServiceTest
	for i := 0; i < numJobs; i++ {
		jobData := MedicalRecordData{ // MedicalRecordData is an alias for dtos.CreateMedicalRecordRequest
			PatientID:  uuid.New(),
			RecordData: json.RawMessage(`{"detail":"record ` + string(rune(i)) + `"}`),
		}
		// Use a sub-context with timeout for ProcessMedicalRecordData to avoid indefinite blocking if jobChan is full
		sendCtx, sendCancel := context.WithTimeout(ctx, 100*time.Millisecond)
		err := svc.ProcessMedicalRecordData(sendCtx, jobData)
		if err != nil {
			t.Errorf("ProcessMedicalRecordData() error = %v for job %d", err, i)
		}
		sendCancel()
	}

	time.Sleep(600 * time.Millisecond) // Give time for processing

	// Use a new context for Stop, as the main 'ctx' might be tied to the overall test timeout.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svc.Stop(stopCtx) 
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	// Assuming processJob calls ListAll for each job.
	if processedCount < int32(numJobs) {
		t.Errorf("Expected at least %d calls to ListAll (from processJob), got %d", numJobs, processedCount)
	}
	t.Logf("ListAll call count for MedicalRecordService: %d (Expected at least %d from jobs)", processedCount, numJobs)
}

// TestMedicalRecordService_Start_ContextCancellation
func TestMedicalRecordService_Start_ContextCancellation(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-ctxcancel: ", log.LstdFlags)
	svc := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)

	ctxService, cancelServiceCtx := context.WithCancel(context.Background())

	err := svc.Start(ctxService)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	go func() {
		jobData := MedicalRecordData{PatientID: uuid.New(), RecordData: json.RawMessage(`{"detail":"cancel test"}`)}
		// Use a separate context or the main test context for sending
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer sendCancel()
		// Ignore error here, as the channel might close during send if cancellation is fast
		_ = svc.ProcessMedicalRecordData(sendCtx, jobData) 
	}()
	
	time.Sleep(20 * time.Millisecond) // Brief pause for the job to be potentially picked up

	cancelServiceCtx() // Cancel the service's context

	// Check if Stop can be called and completes quickly
	stopCompleted := make(chan bool)
	go func() {
		stopCtx, stopCtxCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer stopCtxCancel()
		svc.Stop(stopCtx) // Should be idempotent or handle already closing state
		close(stopCompleted)
	}()

	select {
	case <-stopCompleted:
		t.Log("MedicalRecordService stopped successfully after context cancellation.")
	case <-time.After(3 * time.Second): // Test timeout
		t.Errorf("MedicalRecordService did not stop in time after context cancellation.")
	}
}
