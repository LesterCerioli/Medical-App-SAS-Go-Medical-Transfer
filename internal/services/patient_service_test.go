package services

import (
	"context"
	// "errors" // No longer needed as mock definitions are in mocks_test.go
	"fmt"    
	"log"
	"os"
	"strings" 
	"sync/atomic" 
	"testing"
	"time"

	// "medical-record-service/internal/domain/entities" // REMOVED
	// "medical-record-service/internal/domain/repositories" // REMOVED
	// "github.com/google/uuid" // REMOVED
	"github.com/stretchr/testify/assert"
)

// Note: MockPatientRepository definition and its methods have been moved to mocks_test.go

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
	
	time.Sleep(time.Duration(numJobs/svcImpl.numWorkers+1)*150*time.Millisecond + 200*time.Millisecond)

	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, int32(numJobs), processedCount, "Expected all initial jobs to be processed before stop")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() should not return an error")

	var errAfterStop error
	receivedExpectedError := false
	expectedErrorMessage := "patient service is shutting down, cannot accept new jobs"

	for i := 0; i < 20; i++ { 
		attemptCtx, attemptCancel := context.WithTimeout(context.Background(), 50*time.Millisecond) 
		dummyJobAfterStop := PatientData{Name: "After Stop Job"}
		
		errAfterStop = svcImpl.ProcessPatientData(attemptCtx, dummyJobAfterStop)
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
		t.Errorf("ProcessPatientData after Stop did not return an error after multiple retries, last error: %v", errAfterStop)
	} else if errAfterStop == nil { 
        t.Errorf("ProcessPatientData after Stop returned nil, expected specific error '%s'", expectedErrorMessage)
    } else if !strings.Contains(errAfterStop.Error(), expectedErrorMessage) { 
        t.Errorf("ProcessPatientData after Stop returned error '%v', but expected to contain '%s'", errAfterStop, expectedErrorMessage)
    }
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

	cancelServiceCtx() 

	time.Sleep(200 * time.Millisecond) 

	processedCountBeforeCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	t.Logf("Jobs processed by PatientService before/during cancellation: %d", processedCountBeforeCancel)
	assert.True(t, processedCountBeforeCancel <= int32(numInitialJobs), "Should process at most the initial jobs")

	// Polling loop to check if ProcessPatientData correctly returns error after context cancellation
	var errAfterCtxCancel error
	receivedExpectedErrorAfterCtxCancel := false
	expectedErrorMessageAfterCtxCancel := "patient service is shutting down, cannot accept new jobs"

	for i := 0; i < 20; i++ { // Try up to 20 times
		attemptCtx, attemptCancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // Short timeout for each attempt
		dummyJobAfterCtxCancel := PatientData{Name: "Post Cancel Job"}

		errAfterCtxCancel = svcImpl.ProcessPatientData(attemptCtx, dummyJobAfterCtxCancel)
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
        t.Errorf("ProcessPatientData after context cancellation did not return an error after multiple retries, last error: %v", errAfterCtxCancel)
    } else if errAfterCtxCancel == nil {
        t.Errorf("ProcessPatientData after context cancellation returned nil, expected specific error '%s'", expectedErrorMessageAfterCtxCancel)
    } else if !strings.Contains(errAfterCtxCancel.Error(), expectedErrorMessageAfterCtxCancel) {
        t.Errorf("ProcessPatientData after context cancellation returned error '%v', but expected to contain '%s'", errAfterCtxCancel, expectedErrorMessageAfterCtxCancel)
    }

	time.Sleep(200 * time.Millisecond) 
	processedCountAfterCancel := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	assert.Equal(t, processedCountBeforeCancel, processedCountAfterCancel, "No new jobs should be processed after context cancellation")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	err = svcImpl.Stop(stopCtx)
	assert.NoError(t, err, "Stop() after context cancellation should not error")
}
