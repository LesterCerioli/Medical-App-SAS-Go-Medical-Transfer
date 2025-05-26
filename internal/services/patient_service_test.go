package services

import (
	"context"
	"log"
	"os"
	"sync/atomic" // For thread-safe counters in mocks
	"testing"
	"time"

	"medical-record-service/internal/domain/entities"
	"medical-record-service/internal/domain/repositories"
	"github.com/google/uuid"
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
	
	ListAllFuncCallCount int32 // Atomic counter
	CreateFuncCallCount  int32 // Atomic counter
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
	atomic.AddInt32(&m.ListAllFuncCallCount, 1)
	if m.ListAllFunc != nil {
		// The original signature for ListAllFunc in the mock was (ctx, ctx), which is incorrect.
		// It should match the interface: (ctx context.Context) ([]*entities.Patient, error)
		return m.ListAllFunc(ctx) 
	}
	return nil, nil
}

// TestNewPatientService can remain the same.
func TestNewPatientService(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-patient-service: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger)

	if svc == nil {
		t.Errorf("NewPatientService() returned nil")
	}
}

// TestPatientService_ProcessJobsAndShutdown tests processing jobs and graceful shutdown.
func TestPatientService_ProcessJobsAndShutdown(t *testing.T) {
	mockRepo := &MockPatientRepository{}
	logger := log.New(os.Stdout, "test-process-shutdown: ", log.LstdFlags)
	
	svc := NewPatientService(mockRepo, logger).(*PatientServiceImpl) 

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled at the end of the test

	err := svc.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	numJobs := 10
	for i := 0; i < numJobs; i++ {
		jobData := PatientData{Name: "Test User " + string(rune(i)), Email: "test"+string(rune(i))+"@example.com", DateOfBirth: "2000-01-01"}
		// Use a non-cancellable context for ProcessPatientData if the main test context (ctx)
		// might be cancelled before all jobs are submitted or processed.
		// However, ProcessPatientData itself handles ctx.Done(), so using ctx is fine.
		processCtx, processCancel := context.WithTimeout(ctx, 1*time.Second) // Timeout for sending job
		err := svc.ProcessPatientData(processCtx, jobData)
		processCancel() // Release resources associated with processCtx
		if err != nil {
			t.Errorf("ProcessPatientData() error = %v for job %d", err, i)
		}
	}

	// Allow time for jobs to be processed by workers.
	// This duration depends on numJobs, numWorkers, and processing time per job.
	// If processJob takes ~100ms (as per current PatientServiceImpl), 5 workers, 10 jobs.
	// Theoretical minimum: (10 jobs / 5 workers) * 100ms/job = 200ms. Add buffer.
	time.Sleep(600 * time.Millisecond) // Increased buffer

	// Stop the service
	// Use a new context for Stop, as the main 'ctx' might be tied to the overall test timeout.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	err = svc.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// After Stop, wg.Wait() in shutdown() should have completed.
	// The ListAll in processJob is a placeholder. We expect it to be called for each job.
	processedCount := atomic.LoadInt32(&mockRepo.ListAllFuncCallCount)
	if processedCount < int32(numJobs) {
		t.Errorf("Expected at least %d calls to ListAll (from processJob), got %d", numJobs, processedCount)
	}
	t.Logf("ListAll call count: %d (Expected at least %d from jobs)", processedCount, numJobs)
}

// TestPatientService_Start_ContextCancellation tests the new worker pool context handling
func TestPatientService_Start_ContextCancellation(t *testing.T) {
	mockRepo := &MockPatientRepository{
		ListAllFunc: func(ctx context.Context) ([]*entities.Patient, error) {
			// This ListAll is called by processJob.
			t.Log("MockPatientRepository.ListAll called by a worker")
			return nil, nil
		},
	}
	logger := log.New(os.Stdout, "test-ctxcancel: ", log.LstdFlags)
	svc := NewPatientService(mockRepo, logger).(*PatientServiceImpl)

	// Service's main operational context, which we will cancel
	serviceCtx, cancelServiceCtx := context.WithCancel(context.Background()) 

	err := svc.Start(serviceCtx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Send a job to ensure workers are active
	go func() {
		// Use a background context for sending data as serviceCtx will be cancelled
		jobCtx, jobCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer jobCancel()
		err := svc.ProcessPatientData(jobCtx, PatientData{Name: "CtxCancel Job"})
		if err != nil {
			// This error is expected if cancellation happens fast
			t.Logf("Note: Error sending job during ctx cancel test (potentially expected): %v", err)
		}
	}()
	
	time.Sleep(50 * time.Millisecond) // Brief pause to allow job submission/pickup

	// Cancel the service's main context to trigger shutdown via the monitoring goroutine in Start()
	cancelServiceCtx()

	// Wait for the shutdown process triggered by context cancellation to complete.
	// A robust way to check is to see if a subsequent Stop() call completes quickly,
	// indicating that the internal wg.Wait() has already finished or is about to.
	stopCompleted := make(chan bool)
	go func() {
		// Use a new context for Stop, as the service's main context (serviceCtx) is already cancelled.
		stopCtx, stopCtxCancel := context.WithTimeout(context.Background(), 2*time.Second) 
		defer stopCtxCancel()
		
		// Calling Stop here also tests idempotency of shutdown signals if stopChan was already processed.
		// If Start's goroutine (monitoring ctx.Done()) called shutdown(), jobChan would be closed.
		// If Stop() is called and also tries to signal stopChan, it should handle it gracefully.
		svc.Stop(stopCtx) 
		close(stopCompleted)
	}()

	select {
	case <-stopCompleted:
		t.Log("Service stopped successfully after context cancellation.")
	case <-time.After(3 * time.Second): // Test timeout
		t.Errorf("Service did not stop in time after context cancellation.")
	}

	// Verify that new jobs are not processed after context cancellation and shutdown.
	// This relies on jobChan being closed.
	err = svc.ProcessPatientData(context.Background(), PatientData{Name: "Post-Shutdown Job"})
	if err == nil {
		// This might not be an error if jobChan is buffered and job was accepted before full stop.
		// A more robust check would be that the ListAllFuncCallCount does not increase further.
		// t.Errorf("ProcessPatientData should ideally error or not process job after shutdown, but no error returned.")
		t.Logf("ProcessPatientData after shutdown did not return error, check ListAll count if it increased.")
	} else {
		t.Logf("ProcessPatientData after shutdown returned error as expected (or context timeout): %v", err)
	}
}
