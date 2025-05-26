package services

import (
	"context"
	// "errors" // No longer needed as mock definitions are in mocks_test.go
	"fmt"    
	"log"
	"os"
	"sync/atomic" // Still needed for tests to access counters on the mock
	"testing"
	"time"

	// "medical-record-service/internal/domain/entities" // REMOVED
	// "medical-record-service/internal/domain/repositories" // REMOVED
	// "github.com/google/uuid" // REMOVED
	"github.com/stretchr/testify/assert"
)

// Note: MockPatientRepository definition and its methods have been moved to mocks_test.go

func TestNewPatientService(t *testing.T) {
	mockRepo := &MockPatientRepository{} // This will now refer to the mock in mocks_test.go
	logger := log.New(os.Stdout, "test-patient-service: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger)
	assert.NotNil(t, svc, "NewPatientService() should not return nil")
}

func TestPatientService_WorkersProcessJobs_And_GracefulShutdown(t *testing.T) {
	mockRepo := &MockPatientRepository{} // This will now refer to the mock in mocks_test.go
	logger := log.New(os.Stdout, "test-ps-process-shutdown: ", log.LstdFlags)
	
	svcImpl := NewPatientService(mockRepo, logger).(*PatientServiceImpl) 
	assert.NotNil(t, svcImpl, "NewPatientService returned nil or not PatientServiceImpl")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() 

	err := svcImpl.Start(ctx)
	assert.NoError(t, err, "Start() should not return an error")

	numJobs := 3 * svcImpl.numWorkers 
	for i := 0; i < numJobs; i++ {
		jobData := PatientData{ // PatientData is defined in patient_service_contract.go (same package)
			Name:        fmt.Sprintf("Test User %d", i), 
			Email:       fmt.Sprintf("test%d@example.com", i), 
			DateOfBirth: "2000-01-01",
		}
		sendCtx, sendCancel := context.WithTimeout(ctx, 200*time.Millisecond) 
		err := svcImpl.ProcessPatientData(sendCtx, jobData)
		sendCancel() 
		assert.NoError(t, err, "ProcessPatientData() should not error for job %d", i)
	}
	
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
	mockRepo := &MockPatientRepository{} // This will now refer to the mock in mocks_test.go
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
