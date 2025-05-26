package services

import (
	"context"
	"errors" 
	"log"
	"sync" 
	"time"

	"medical-record-service/internal/domain/repositories"
	// MedicalRecordData is defined in medical_record_service_contract.go (same package)
	// and is an alias for dtos.CreateMedicalRecordRequest.
	// The contract file imports dtos, so the type is available.
)

// MedicalRecordServiceImpl implements MedicalRecordServiceContract for medical record data processing.
type MedicalRecordServiceImpl struct {
	medicalRecordRepo repositories.MedicalRecordRepositoryContract
	logger            *log.Logger
	jobChan           chan MedicalRecordData 
	stopChan          chan struct{}          
	numWorkers        int                    
	wg                sync.WaitGroup         
	serviceCtx        context.Context    // For overall service lifecycle
	serviceCancel     context.CancelFunc // To cancel serviceCtx
}

// NewMedicalRecordService creates a new instance of MedicalRecordServiceImpl.
func NewMedicalRecordService(repo repositories.MedicalRecordRepositoryContract, logger *log.Logger) MedicalRecordServiceContract {
	numWorkers := 5 
	ctx, cancel := context.WithCancel(context.Background())
	return &MedicalRecordServiceImpl{
		medicalRecordRepo: repo,
		logger:            logger,
		jobChan:           make(chan MedicalRecordData, 100), 
		stopChan:          make(chan struct{}), 
		numWorkers:        numWorkers,
		serviceCtx:        ctx,    
		serviceCancel:     cancel, 
	}
}

// worker is a goroutine that processes jobs from jobChan.
func (s *MedicalRecordServiceImpl) worker(id int) {
	defer s.wg.Done() 
	s.logger.Printf("Medical Record Worker %d started", id)
	for {
		select {
		case data, ok := <-s.jobChan:
			if !ok { 
				s.logger.Printf("Medical Record Worker %d: jobChan closed, finishing.", id)
				return
			}
			// Assuming MedicalRecordData has PatientID (it's an alias for dtos.CreateMedicalRecordRequest)
			s.logger.Printf("Medical Record Worker %d processing data for patient ID: %s", id, data.PatientID)
			s.processJob(s.serviceCtx, data) 
		case <-s.serviceCtx.Done(): 
			s.logger.Printf("Medical Record Worker %d: service context done, finishing.", id)
			return
		}
	}
}

// processJob contains the actual logic for processing a single MedicalRecordData item.
func (s *MedicalRecordServiceImpl) processJob(ctx context.Context, data MedicalRecordData) {
	s.logger.Printf("Worker processing medical record: %+v", data)
	
	select {
	case <-time.After(100 * time.Millisecond): // Simulate work
		if s.medicalRecordRepo != nil { 
			_, _ = s.medicalRecordRepo.ListAll(ctx) 
		}
		s.logger.Printf("Finished processing medical record for patient ID: %s", data.PatientID)
	case <-ctx.Done(): 
		s.logger.Printf("Job processing cancelled for patient ID: %s", data.PatientID)
		return
	}
}

// Start initializes and starts the worker pool and the service monitoring.
func (s *MedicalRecordServiceImpl) Start(ctx context.Context) error { 
	s.logger.Println("Medical Record service starting with worker pool...")

	s.wg.Add(s.numWorkers)
	for i := 1; i <= s.numWorkers; i++ {
		go s.worker(i)
	}
	s.logger.Printf("%d medical record workers started", s.numWorkers)

	go func() {
		select {
		case <-ctx.Done(): 
			s.logger.Println("Medical Record service parent context cancelled, initiating shutdown...")
			s.shutdown()
		case <-s.stopChan: 
			s.logger.Println("Medical Record service received stop signal, initiating shutdown...")
			s.shutdown()
		}
	}()

	return nil
}

// shutdown is an internal method to gracefully stop all workers.
func (s *MedicalRecordServiceImpl) shutdown() {
	s.logger.Println("Medical Record service initiating shutdown of worker pool...")
	s.serviceCancel() 
	
	s.wg.Wait()      
	s.logger.Println("All medical record workers have finished.")
	
	s.logger.Println("Closing jobChan for medical records.")
	close(s.jobChan) 

	s.logger.Println("Medical Record service shutdown complete.")
}

// Stop signals the service to gracefully shut down.
func (s *MedicalRecordServiceImpl) Stop(ctx context.Context) error {
	s.logger.Println("Medical Record service stop requested...")
	select {
	case s.stopChan <- struct{}{}:
		s.logger.Println("Stop signal sent to medical record service monitoring goroutine.")
	default:
		s.logger.Println("Medical Record service already stopping or stop signal channel full/unmonitored.")
	}
	return nil
}

// ProcessMedicalRecordData sends medical record data to the job channel.
func (s *MedicalRecordServiceImpl) ProcessMedicalRecordData(ctx context.Context, data MedicalRecordData) error {
	s.logger.Printf("Attempting to send medical record data for Patient ID: %s to job queue.", data.PatientID)

	select {
	case <-s.serviceCtx.Done():
		s.logger.Printf("Service context is done. Cannot process new medical record for Patient ID: %s. Error: %v", data.PatientID, s.serviceCtx.Err())
		return errors.New("service is shutting down, cannot accept new medical record jobs")
	default:
	}

	select {
	case s.jobChan <- data:
		s.logger.Printf("Medical record data for Patient ID: %s sent to job queue.", data.PatientID)
		return nil
	case <-ctx.Done(): 
		s.logger.Printf("Request context cancelled for Medical Record (Patient ID: %s): %v", data.PatientID, ctx.Err())
		return ctx.Err()
	case <-s.serviceCtx.Done(): 
		s.logger.Printf("Service context was cancelled while attempting to send Medical Record (Patient ID: %s). Error: %v", data.PatientID, s.serviceCtx.Err())
		return errors.New("service shutting down while sending medical record job")
	}
}
