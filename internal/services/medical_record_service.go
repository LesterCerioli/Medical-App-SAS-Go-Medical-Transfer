package services

import (
	"context"
	"log"
	"sync" // Added for sync.WaitGroup
	"time"

	"medical-record-service/internal/domain/repositories"
	// medical_record_service_contract.go defines MedicalRecordData in the same 'services' package
)

// MedicalRecordServiceImpl implements MedicalRecordServiceContract for medical record data processing.
type MedicalRecordServiceImpl struct {
	medicalRecordRepo repositories.MedicalRecordRepositoryContract
	logger            *log.Logger
	jobChan           chan MedicalRecordData // Channel for incoming medical record data jobs
	stopChan          chan struct{}          // Channel to signal service stop
	numWorkers        int                    // Number of worker goroutines
	wg                sync.WaitGroup         // WaitGroup to manage worker lifecycle
}

// NewMedicalRecordService creates a new instance of MedicalRecordServiceImpl.
func NewMedicalRecordService(repo repositories.MedicalRecordRepositoryContract, logger *log.Logger) MedicalRecordServiceContract {
	numWorkers := 5 // Configurable number of workers
	return &MedicalRecordServiceImpl{
		medicalRecordRepo: repo,
		logger:            logger,
		jobChan:           make(chan MedicalRecordData, 100), // Buffered channel for jobs
		stopChan:          make(chan struct{}),
		numWorkers:        numWorkers,
	}
}

// worker is a goroutine that processes jobs from jobChan.
func (s *MedicalRecordServiceImpl) worker(id int) {
	defer s.wg.Done() // Decrement WaitGroup counter when worker exits
	s.logger.Printf("Medical Record Worker %d started", id)
	for data := range s.jobChan { // Loop continues until jobChan is closed
		// Assuming MedicalRecordData (alias for dtos.CreateMedicalRecordRequest) has PatientID for logging
		s.logger.Printf("Medical Record Worker %d processing data for patient ID: %s", id, data.PatientID)
		s.processJob(context.Background(), data) // Using Background context for now
	}
	s.logger.Printf("Medical Record Worker %d finished", id)
}

// processJob contains the actual logic for processing a single MedicalRecordData item.
func (s *MedicalRecordServiceImpl) processJob(ctx context.Context, data MedicalRecordData) {
	s.logger.Printf("Worker processing medical record: %+v", data)
	time.Sleep(100 * time.Millisecond) // Simulate work

	// Important: Add a placeholder call to a method of medicalRecordRepo
	if s.medicalRecordRepo != nil { // Check if repo is initialized
		_, _ = s.medicalRecordRepo.ListAll(ctx) // Example call to avoid "unused" error
	}

	s.logger.Printf("Finished processing medical record for patient ID: %s", data.PatientID)
}

// Start initializes and starts the worker pool and the service monitoring.
func (s *MedicalRecordServiceImpl) Start(ctx context.Context) error {
	s.logger.Println("Medical Record service starting with worker pool...")

	s.wg.Add(s.numWorkers)
	for i := 1; i <= s.numWorkers; i++ {
		go s.worker(i)
	}
	s.logger.Printf("%d medical record workers started", s.numWorkers)

	// Goroutine to handle graceful shutdown on context cancellation or explicit Stop
	go func() {
		select {
		case <-ctx.Done(): // Context from higher up (e.g., application lifecycle) is cancelled
			s.logger.Println("Medical Record service context cancelled, initiating shutdown...")
			s.shutdown()
		case <-s.stopChan: // Explicit Stop() call
			s.logger.Println("Medical Record service received stop signal, initiating shutdown...")
			s.shutdown()
		}
	}()

	return nil
}

// shutdown is an internal method to gracefully stop all workers.
func (s *MedicalRecordServiceImpl) shutdown() {
	s.logger.Println("Medical Record service initiating shutdown of worker pool...")
	close(s.jobChan) // Close jobChan to signal workers to stop processing new jobs
	s.wg.Wait()      // Wait for all worker goroutines to finish their current jobs and exit
	s.logger.Println("All medical record workers have finished.")
}

// Stop signals the service to gracefully shut down.
func (s *MedicalRecordServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("Medical Record service stop requested...")
	// Using a non-blocking send to stopChan to prevent deadlock if already stopping or stopped.
	select {
	case s.stopChan <- struct{}{}:
		s.logger.Println("Stop signal sent to medical record service.")
	default:
		s.logger.Println("Medical Record service already stopping or stop signal channel full.")
	}
	return nil
}

// ProcessMedicalRecordData sends medical record data to the job channel for asynchronous processing.
func (s *MedicalRecordServiceImpl) ProcessMedicalRecordData(ctx context.Context, data MedicalRecordData) error {
	s.logger.Printf("Received medical record data for processing (Patient ID: %s)", data.PatientID)

	select {
	case s.jobChan <- data:
		s.logger.Println("Medical record data sent to job queue.")
		return nil
	case <-ctx.Done():
		s.logger.Printf("Context cancelled, could not send medical record data (Patient ID: %s) to job queue: %v", data.PatientID, ctx.Err())
		return ctx.Err()
	}
}
