package services

import (
	"context"
	"log"
	"sync" // Added for sync.WaitGroup
	"time"

	"medical-record-service/internal/domain/repositories"
	// patient_service_contract.go defines PatientData in the same 'services' package
)

// PatientServiceImpl implements PatientServiceContract for patient data processing.
type PatientServiceImpl struct {
	patientRepo repositories.PatientRepositoryContract
	logger      *log.Logger
	jobChan     chan PatientData  // Channel for incoming patient data jobs
	stopChan    chan struct{}     // Channel to signal service stop
	numWorkers  int               // Number of worker goroutines
	wg          sync.WaitGroup    // WaitGroup to manage worker lifecycle
}

// NewPatientService creates a new instance of PatientServiceImpl.
func NewPatientService(repo repositories.PatientRepositoryContract, logger *log.Logger) PatientServiceContract {
	numWorkers := 5 // Configurable number of workers
	return &PatientServiceImpl{
		patientRepo: repo,
		logger:      logger,
		jobChan:     make(chan PatientData, 100), // Buffered channel for jobs
		stopChan:    make(chan struct{}),
		numWorkers:  numWorkers,
	}
}

// worker is a goroutine that processes jobs from jobChan.
func (s *PatientServiceImpl) worker(id int) {
	defer s.wg.Done() // Decrement WaitGroup counter when worker exits
	s.logger.Printf("Worker %d started", id)
	for data := range s.jobChan { // Loop continues until jobChan is closed
		s.logger.Printf("Worker %d processing data for patient: %s", id, data.Name) // Assuming PatientData has Name
		s.processJob(context.Background(), data) // Using Background context for now
	}
	s.logger.Printf("Worker %d finished", id)
}

// processJob contains the actual logic for processing a single PatientData item.
func (s *PatientServiceImpl) processJob(ctx context.Context, data PatientData) {
	s.logger.Printf("Processing job for patient: %s, Email: %s", data.Name, data.Email) // Example logging
	
	// Simulate work
	time.Sleep(100 * time.Millisecond)

	// Placeholder: Example of using the repository to avoid "unused" errors.
	// In a real scenario, this would be a meaningful database operation.
	// For instance, creating or updating a patient record based on 'data'.
	// _, err := s.patientRepo.FindByEmail(ctx, data.Email) // Example find
	// if err != nil {
	// 	s.logger.Printf("Error in processJob (placeholder repo call): %v", err)
	// }
	// For now, to ensure patientRepo is "used" if no direct repo call is made with `data`:
	if s.patientRepo != nil { // Check if repo is initialized (it should be)
		_, _ = s.patientRepo.ListAll(ctx) // Generic call, adjust as needed
	}

	s.logger.Printf("Finished processing job for patient: %s", data.Name)
}

// Start initializes and starts the worker pool and the service monitoring.
func (s *PatientServiceImpl) Start(ctx context.Context) error {
	s.logger.Println("Patient service starting with worker pool...")

	s.wg.Add(s.numWorkers)
	for i := 1; i <= s.numWorkers; i++ {
		go s.worker(i)
	}
	s.logger.Printf("%d workers started", s.numWorkers)

	// Goroutine to handle graceful shutdown on context cancellation or explicit Stop
	go func() {
		select {
		case <-ctx.Done(): // Context from higher up (e.g., application lifecycle) is cancelled
			s.logger.Println("Patient service context cancelled, initiating shutdown...")
			s.shutdown()
		case <-s.stopChan: // Explicit Stop() call
			s.logger.Println("Patient service received stop signal, initiating shutdown...")
			s.shutdown()
		}
	}()

	return nil
}

// shutdown is an internal method to gracefully stop all workers.
func (s *PatientServiceImpl) shutdown() {
	s.logger.Println("Patient service initiating shutdown of worker pool...")
	close(s.jobChan) // Close jobChan to signal workers to stop processing new jobs
	s.wg.Wait()      // Wait for all worker goroutines to finish their current jobs and exit
	s.logger.Println("All patient workers have finished.")
	// Note: stopChan is not closed here as it's used to signal this shutdown func.
	// If stopChan was only for external stop signal, it could be closed in Stop().
}

// Stop signals the service to gracefully shut down.
func (s *PatientServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("Patient service stop requested...")
	// Signal the monitoring goroutine in Start (if it's running) or directly shutdown.
	// Using a non-blocking send to stopChan to prevent deadlock if already stopping or stopped.
	select {
	case s.stopChan <- struct{}{}:
		s.logger.Println("Stop signal sent to patient service.")
	default:
		s.logger.Println("Patient service already stopping or stop signal channel full.")
	}
	// It's important that shutdown() is idempotent or handled correctly if called multiple times.
	// The current structure relies on the Start goroutine to call shutdown.
	// If Start's goroutine might not be running (e.g., Start not called),
	// a direct call to shutdown() here might be considered, but needs careful state management.
	return nil
}

// ProcessPatientData sends patient data to the job channel for asynchronous processing by a worker.
func (s *PatientServiceImpl) ProcessPatientData(ctx context.Context, data PatientData) error {
	s.logger.Printf("Received patient data for processing: Name %s", data.Name)

	// Using a select to make the send non-blocking or handle context cancellation.
	select {
	case s.jobChan <- data:
		s.logger.Printf("Patient data for %s sent to job queue.", data.Name)
		return nil
	case <-ctx.Done():
		s.logger.Printf("Context cancelled while trying to send patient data for %s to queue: %v", data.Name, ctx.Err())
		return ctx.Err()
	// Optional: Timeout for sending to queue
	// case <-time.After(1 * time.Second):
	// 	s.logger.Printf("Timeout sending patient data for %s to queue.", data.Name)
	// 	return errors.New("timeout sending data to job queue")
	}
}
