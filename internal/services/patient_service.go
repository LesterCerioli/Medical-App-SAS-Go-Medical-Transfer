package services

import (
	"context"
	"errors" // Added for errors.New
	"log"
	"sync" 
	"time"

	"medical-record-service/internal/domain/repositories"
	// PatientData is defined in patient_service_contract.go (same package)
	// and is an alias for dtos.CreatePatientRequest.
	// The contract file imports dtos, so the type is available.
)

// PatientServiceImpl implements PatientServiceContract for patient data processing.
type PatientServiceImpl struct {
	patientRepo repositories.PatientRepositoryContract
	logger      *log.Logger
	jobChan     chan PatientData  
	stopChan    chan struct{}     
	numWorkers  int               
	wg          sync.WaitGroup    
	serviceCtx  context.Context    // For overall service lifecycle
	serviceCancel context.CancelFunc // To cancel serviceCtx
}

// NewPatientService creates a new instance of PatientServiceImpl.
func NewPatientService(repo repositories.PatientRepositoryContract, logger *log.Logger) PatientServiceContract {
	numWorkers := 5 
	ctx, cancel := context.WithCancel(context.Background())
	return &PatientServiceImpl{
		patientRepo: repo,
		logger:      logger,
		jobChan:     make(chan PatientData, 100), 
		stopChan:    make(chan struct{}),
		numWorkers:  numWorkers,
		serviceCtx:  ctx,
		serviceCancel: cancel,
	}
}

// worker is a goroutine that processes jobs from jobChan.
func (s *PatientServiceImpl) worker(id int) {
	defer s.wg.Done() 
	s.logger.Printf("Patient Worker %d started", id)
	for {
		select {
		case data, ok := <-s.jobChan:
			if !ok { 
				s.logger.Printf("Patient Worker %d: jobChan closed, finishing.", id)
				return
			}
			s.logger.Printf("Patient Worker %d processing data for patient: %s", id, data.Name) 
			s.processJob(s.serviceCtx, data) // Pass serviceCtx so job respects service shutdown
		case <-s.serviceCtx.Done(): 
			s.logger.Printf("Patient Worker %d: service context done, finishing.", id)
			return
		}
	}
}

// processJob contains the actual logic for processing a single PatientData item.
func (s *PatientServiceImpl) processJob(ctx context.Context, data PatientData) {
	s.logger.Printf("Processing job for patient: %s, Email: %s", data.Name, data.Email) 
	
	select {
	case <-time.After(100 * time.Millisecond): // Simulate work
		if s.patientRepo != nil { 
			_, _ = s.patientRepo.ListAll(ctx) 
		}
		s.logger.Printf("Finished processing job for patient: %s", data.Name)
	case <-ctx.Done(): // If the job's context (s.serviceCtx) is cancelled
		s.logger.Printf("Job processing cancelled for patient: %s", data.Name)
		return
	}
}

// Start initializes and starts the worker pool and the service monitoring.
func (s *PatientServiceImpl) Start(ctx context.Context) error { 
	s.logger.Println("Patient service starting with worker pool...")

	s.wg.Add(s.numWorkers)
	for i := 1; i <= s.numWorkers; i++ {
		go s.worker(i)
	}
	s.logger.Printf("%d patient workers started", s.numWorkers)

	go func() {
		select {
		case <-ctx.Done(): 
			s.logger.Println("Patient service parent context cancelled, initiating shutdown...")
			s.shutdown()
		case <-s.stopChan: 
			s.logger.Println("Patient service received stop signal, initiating shutdown...")
			s.shutdown()
		}
	}()

	return nil
}

// shutdown is an internal method to gracefully stop all workers.
func (s *PatientServiceImpl) shutdown() {
	s.logger.Println("Patient service initiating shutdown of worker pool...")
	s.serviceCancel() // 1. Signal all operations using serviceCtx to cancel (workers, job processing)
	
	s.wg.Wait()      
	s.logger.Println("All patient workers have finished.")
	
	s.logger.Println("Closing jobChan for patients.")
	close(s.jobChan) // Workers will see channel closed and exit their range loop.

	s.logger.Println("Patient service shutdown complete.")
}

// Stop signals the service to gracefully shut down.
func (s *PatientServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("Patient service stop requested...")
	select {
	case s.stopChan <- struct{}{}:
		s.logger.Println("Stop signal sent to patient service monitoring goroutine.")
	default:
		s.logger.Println("Patient service already stopping or stop signal channel full/unmonitored.")
	}
	return nil
}

// ProcessPatientData sends patient data to the job channel for asynchronous processing by a worker.
func (s *PatientServiceImpl) ProcessPatientData(ctx context.Context, data PatientData) error {
	s.logger.Printf("Received patient data for processing: %+v", data) // Changed to %+v for more detail
	select {
	case <-s.serviceCtx.Done():
		s.logger.Println("Patient service is shutting down, cannot accept new jobs.")
		return errors.New("patient service is shutting down, cannot accept new jobs")
	default:
		// Continue if service context is not done
	}

	select {
	case s.jobChan <- data:
		s.logger.Println("Patient data sent to job queue.")
		return nil
	case <-ctx.Done(): // Contexto da requisição específica
		s.logger.Printf("Context for sending patient data cancelled: %v", ctx.Err())
		return ctx.Err()
	// No default here, so it blocks if jobChan is full, until ctx is done.
	// Alternative:
	// case <-time.After(1*time.Second): // Non-blocking with timeout
	//  s.logger.Println("Timeout sending patient data to job queue.")
	//  return errors.New("job queue full or send timeout")
	}
}
