package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Compile-time check to ensure MockMedicalRecordRepository implements MedicalRecordRepositoryContract
var _ repositories.MedicalRecordRepositoryContract = (*MockMedicalRecordRepository)(nil)

// MockMedicalRecordRepository is a mock implementation of MedicalRecordRepositoryContract.
type MockMedicalRecordRepository struct {
	CreateFunc          func(ctx context.Context, record *entities.MedicalRecord) error
	GetByIDFunc         func(ctx context.Context, id uuid.UUID) (*entities.MedicalRecord, error)
	UpdateFunc          func(ctx context.Context, record *entities.MedicalRecord) error
	DeleteFunc          func(ctx context.Context, id uuid.UUID) error
	FindByPatientIDFunc func(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicalRecord, error)
	ListAllFunc         func(ctx context.Context) ([]*entities.MedicalRecord, error)

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

	svcImpl := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)
	assert.NotNil(t, svcImpl, "NewMedicalRecordService returned nil or not MedicalRecordServiceImpl")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := svcImpl.Start(ctx)
	assert.NoError(t, err, "Start() should not return an error")

	numJobs := 3 * svcImpl.numWorkers
	for i := 0; i < numJobs; i++ {
		jobData := MedicalRecordData{
			PatientID:  uuid.New(),
			RecordData: json.RawMessage(fmt.Sprintf(`{"detail":"record %d"}`, i)),
		}
		sendCtx, sendCancel := context.WithTimeout(ctx, 200*time.Millisecond)
		err := svcImpl.ProcessMedicalRecordData(sendCtx, jobData)
		sendCancel()
		assert.NoError(t, err, "ProcessMedicalRecordData() should not error for job %d", i)
	}

	time.Sleep(time.Duration(numJobs/svcImpl.numWorkers+1)*150*time.Millisecond + 200*time.Millisecond)

	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, int32(numJobs), processedCount, "Expected all initial jobs to be processed before stop")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() should not return an error")

	var errAfterStop error
	receivedExpectedError := false
	expectedErrorMessage := "service is shutting down, cannot accept new medical record jobs"

	for i := 0; i < 20; i++ {
		attemptCtx, attemptCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		dummyJobAfterStop := MedicalRecordData{PatientID: uuid.New(), RecordData: json.RawMessage(`{}`)}

		errAfterStop = svcImpl.ProcessMedicalRecordData(attemptCtx, dummyJobAfterStop)
		attemptCancel()

		if errAfterStop != nil {
			if strings.Contains(errAfterStop.Error(), expectedErrorMessage) {
				receivedExpectedError = true
				t.Logf("Correctly received error on attempt %d after Stop: %v", i+1, errAfterStop)
				break
			}
			t.Logf("Received an unexpected error on attempt %d after Stop: %v", i+1, errAfterStop)
			receivedExpectedError = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !receivedExpectedError {
		t.Errorf("ProcessMedicalRecordData after Stop did not return an error after multiple retries, last error: %v", errAfterStop)
	} else if errAfterStop == nil {
		t.Errorf("ProcessMedicalRecordData after Stop returned nil, but an error was expected (loop logic issue)")
	} else if !strings.Contains(errAfterStop.Error(), expectedErrorMessage) {
		t.Errorf("ProcessMedicalRecordData after Stop returned error '%v', but expected to contain '%s'", errAfterStop, expectedErrorMessage)
	}
}

func TestMedicalRecordService_Start_ContextCancellation_StopsProcessing(t *testing.T) {
	mockRepo := &MockMedicalRecordRepository{}
	logger := log.New(os.Stdout, "test-mr-ctxcancel: ", log.LstdFlags)
	svcImpl := NewMedicalRecordService(mockRepo, logger).(*MedicalRecordServiceImpl)

	serviceCtx, cancelServiceCtx := context.WithCancel(context.Background())

	err := svcImpl.Start(serviceCtx)
	assert.NoError(t, err, "Start() should not return an error")

	numInitialJobs := svcImpl.numWorkers
	for i := 0; i < numInitialJobs; i++ {
		jobData := MedicalRecordData{PatientID: uuid.New(), RecordData: json.RawMessage(`{"detail":"initial job"}`)}
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_ = svcImpl.ProcessMedicalRecordData(sendCtx, jobData)
		sendCancel()
	}

	time.Sleep(time.Duration(numInitialJobs/svcImpl.numWorkers+1) * 50 * time.Millisecond)

	cancelServiceCtx()

	time.Sleep(200 * time.Millisecond)

	processedCountBeforeCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	t.Logf("Jobs processed before/during cancellation: %d", processedCountBeforeCancel)
	assert.True(t, processedCountBeforeCancel <= int32(numInitialJobs), "Should process at most the initial jobs")

	// Polling loop to check if ProcessMedicalRecordData correctly returns error after context cancellation
	var errAfterCtxCancel error
	receivedExpectedErrorAfterCtxCancel := false
	expectedErrorMessageAfterCtxCancel := "service is shutting down, cannot accept new medical record jobs"

	for i := 0; i < 20; i++ { // Try up to 20 times
		// Using context.Background() directly as the attempt itself should be quick
		// and not rely on the now-cancelled serviceCtx for its own execution.
		attemptCtx, attemptCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		dummyJobAfterCtxCancel := MedicalRecordData{PatientID: uuid.New(), RecordData: json.RawMessage(`{"detail":"post-cancel job"}`)}

		errAfterCtxCancel = svcImpl.ProcessMedicalRecordData(attemptCtx, dummyJobAfterCtxCancel)
		attemptCancel()

		if errAfterCtxCancel != nil {
			if strings.Contains(errAfterCtxCancel.Error(), expectedErrorMessageAfterCtxCancel) {
				receivedExpectedErrorAfterCtxCancel = true
				t.Logf("Correctly received error on attempt %d after context cancellation: %v", i+1, errAfterCtxCancel)
				break
			}
			t.Logf("Received an unexpected error on attempt %d after context cancellation: %v", i+1, errAfterCtxCancel)
			receivedExpectedErrorAfterCtxCancel = true // Mark to break, assertions below will check specific error
			break
		}
		time.Sleep(100 * time.Millisecond) // Wait before retrying
	}

	if !receivedExpectedErrorAfterCtxCancel {
		t.Errorf("ProcessMedicalRecordData after context cancellation did not return an error after multiple retries, last error: %v", errAfterCtxCancel)
	} else if errAfterCtxCancel == nil {
		t.Errorf("ProcessMedicalRecordData after context cancellation returned nil, expected specific error '%s'", expectedErrorMessageAfterCtxCancel)
	} else if !strings.Contains(errAfterCtxCancel.Error(), expectedErrorMessageAfterCtxCancel) {
		t.Errorf("ProcessMedicalRecordData after context cancellation returned error '%v', but expected to contain '%s'", errAfterCtxCancel, expectedErrorMessageAfterCtxCancel)
	}

	time.Sleep(200 * time.Millisecond)
	processedCountAfterCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, processedCountBeforeCancel, processedCountAfterCancel, "No new jobs should be processed after context cancellation")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() after context cancellation should not error")
}
